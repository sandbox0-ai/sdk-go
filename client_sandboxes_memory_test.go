package sandbox0

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestExecutionStateOptionsPreserveDefaultAndRejectFallback(t *testing.T) {
	for _, action := range []string{"pause", "resume"} {
		for _, tc := range []struct {
			name    string
			options *SandboxExecutionStateOptions
			failed  bool
		}{
			{name: "legacy"},
			{name: "filesystem", options: &SandboxExecutionStateOptions{}},
			{name: "memory", options: &SandboxExecutionStateOptions{Memory: true}},
			{name: "unsupported", options: &SandboxExecutionStateOptions{Memory: true}, failed: true},
		} {
			t.Run(action+"/"+tc.name, func(t *testing.T) {
				var calls atomic.Int32
				client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Path != "/api/v1/sandboxes/sb_1/"+action {
						t.Fatalf("unexpected path %s", r.URL.Path)
					}
					raw, err := io.ReadAll(r.Body)
					if err != nil {
						t.Fatal(err)
					}
					if tc.options == nil {
						if len(raw) != 0 {
							t.Fatalf("legacy request sent body %q", raw)
						}
					} else {
						var body map[string]any
						if err := json.Unmarshal(raw, &body); err != nil {
							t.Fatal(err)
						}
						if body["memory"] != tc.options.Memory {
							t.Fatalf("body = %s", raw)
						}
					}
					if tc.failed {
						writeJSON(t, w, http.StatusServiceUnavailable, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "memory unavailable"}})
						return
					}
					writeJSON(t, w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"sandbox_id": "sb_1", "paused": true, "status": "paused", "resumed": true}})
				})
				defer server.Close()
				var err error
				if action == "pause" {
					if tc.options == nil {
						_, err = client.PauseSandbox(t.Context(), "sb_1")
					} else {
						_, err = client.PauseSandboxWithOptions(t.Context(), "sb_1", tc.options)
					}
				} else {
					if tc.options == nil {
						_, err = client.ResumeSandbox(t.Context(), "sb_1")
					} else {
						_, err = client.ResumeSandboxWithOptions(t.Context(), "sb_1", tc.options)
					}
				}
				if tc.failed {
					var apiErr *APIError
					if !errors.As(err, &apiErr) || apiErr.StatusCode != 503 {
						t.Fatalf("error = %v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
				if calls.Load() != 1 {
					t.Fatalf("requests = %d; must not retry with filesystem defaults", calls.Load())
				}
			})
		}
	}
}

func TestMemoryForkRetriesWithExplicitStableKey(t *testing.T) {
	var calls atomic.Int32
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		if r.Header.Get("Idempotency-Key") != "memory-fork-one" {
			t.Fatalf("missing stable key: %v", r.Header)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["memory"] != true {
			t.Fatalf("memory lost on retry: %v", body)
		}
		if call == 1 {
			writeJSON(t, w, 503, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "memory fork capture is pending"}})
			return
		}
		writeJSON(t, w, 201, map[string]any{"success": true, "data": map[string]any{"source_sandbox_id": "sb_source", "sandbox": sandboxJSON("sb_child")}})
	})
	defer server.Close()
	request := &apispec.ForkSandboxRequest{Memory: apispec.NewOptBool(true)}
	if _, err := client.ForkSandbox(context.Background(), "sb_source", request); err == nil {
		t.Fatal("missing key accepted")
	}
	if calls.Load() != 0 {
		t.Fatal("missing key reached API")
	}
	options := &ForkSandboxOptions{IdempotencyKey: "memory-fork-one"}
	_, err := client.ForkSandboxWithOptions(t.Context(), "sb_source", request, options)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 503 {
		t.Fatalf("pending error = %v", err)
	}
	result, err := client.ForkSandboxWithOptions(t.Context(), "sb_source", request, options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Sandbox.ID != "sb_child" || calls.Load() != 2 {
		t.Fatalf("result = %v, calls = %d", result, calls.Load())
	}
}

func TestMemoryPauseWaitRejectsFailedCaptureWithoutColdFallback(t *testing.T) {
	posts, gets := 0, 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["memory"] != true {
				t.Fatalf("unexpected pause: %v %v", body, err)
			}
			writeJSON(t, w, http.StatusAccepted, map[string]any{"success": true, "data": map[string]any{"sandbox_id": "sb_123", "paused": false, "status": "starting"}})
			return
		}
		gets++
		payload := sandboxJSON("sb_123")
		payload["status"], payload["paused"] = "failed", true
		writeJSON(t, w, http.StatusOK, map[string]any{"success": true, "data": payload})
	})
	defer server.Close()
	got, err := client.PauseSandboxAndWait(context.Background(), "sb_123", &SandboxLifecycleWaitOptions{Memory: true})
	var failed *SandboxLifecycleFailedError
	if got != nil || !errors.As(err, &failed) || failed.LastSandbox == nil || !failed.LastSandbox.Paused || posts != 1 || gets != 1 {
		t.Fatalf("memory capture failure was not returned exactly: %v %v posts=%d gets=%d", got, err, posts, gets)
	}
}

func TestMemoryResumeWaitReportsNewFailureWithoutRetryingExecution(t *testing.T) {
	posts, gets := 0, 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
			writeJSON(t, w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"sandbox_id": "sb_123", "resumed": true}})
			return
		}
		gets++
		payload := sandboxJSON("sb_123")
		payload["status"], payload["paused"], payload["runtime_generation"] = "failed", true, 1
		if gets >= 3 {
			payload["runtime_generation"] = 2
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"success": true, "data": payload})
	})
	defer server.Close()
	got, err := client.ResumeSandboxAndWait(context.Background(), "sb_123", &SandboxLifecycleWaitOptions{Memory: true, PollInterval: time.Millisecond})
	var failed *SandboxLifecycleFailedError
	if got != nil || !errors.As(err, &failed) || failed.Action != "memory resume" || failed.LastSandbox.RuntimeGeneration != 2 || posts != 1 || gets != 3 {
		t.Fatalf("memory restore failure not returned exactly: %v %v posts=%d gets=%d", got, err, posts, gets)
	}
}
