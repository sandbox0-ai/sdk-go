package sandbox0

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

var forkLaunchOptions = RootFSForkLaunchOptions{ForkOperationID: "fork", FilesystemID: "child", HeadGenerationID: "head", WriterEpoch: 7, IdempotencyKey: "original", ExpectedOperationID: "launch"}

func forkLaunchFixture(completed bool) map[string]any {
	r := map[string]any{"operation_id": "launch", "sandbox_id": "child", "fork_operation_id": "fork", "filesystem_id": "child", "head_generation_id": "head", "writer_epoch": 7, "lifecycle_epoch": 1, "runtime_generation": 0, "state": "pending", "current": true}
	if completed {
		r["state"] = "completed"
		r["runtime_generation"] = 1
		r["current"] = false
		r["launch_digest"] = strings.Repeat("a", 64)
		r["completed_at"] = "2026-10-17T00:00:00Z"
	}
	return r
}

func callForkLaunch(t *testing.T, c *Client, method string, q RootFSForkLaunchOptions) (*apispec.SandboxRootFSForkLaunch, error) {
	t.Helper()
	if method == "POST" {
		return c.LaunchSandboxRootFSFork(t.Context(), "child", q)
	}
	return c.GetSandboxRootFSForkLaunch(t.Context(), "child", q)
}

func TestForkLaunchOriginalConditionsAndOutcomes(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		for _, status := range []int{200, 202, 404, 409, 429, 503} {
			if method == "GET" && status == 202 {
				continue
			}
			t.Run(fmt.Sprintf("%s-%d", method, status), func(t *testing.T) {
				calls := 0
				c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Method != method || r.URL.Path != "/api/v1/sandboxes/child/fork-launch" || r.Header.Get("Idempotency-Key") != "original" {
						t.Error("original request changed")
					}
					if method == "POST" {
						var body map[string]any
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Fatal(err)
						}
						if len(body) != 4 || body["fork_operation_id"] != "fork" || body["filesystem_id"] != "child" || body["head_generation_id"] != "head" || body["writer_epoch"] != float64(7) || r.URL.RawQuery != "" {
							t.Error("original body changed")
						}
					} else {
						q := r.URL.Query()
						if len(q) != 4 || q.Get("fork_operation_id") != "fork" || q.Get("filesystem_id") != "child" || q.Get("head_generation_id") != "head" || q.Get("writer_epoch") != "7" || r.ContentLength > 0 {
							t.Error("original query changed")
						}
					}
					if status >= 400 {
						writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]any{"code": "retained", "message": "retain original"}})
						return
					}
					writeJSON(t, w, status, map[string]any{"success": true, "data": forkLaunchFixture(status == 200)})
				})
				defer server.Close()
				r, err := callForkLaunch(t, c, method, forkLaunchOptions)
				if status >= 400 {
					var apiErr *APIError
					if !errors.As(err, &apiErr) || apiErr.StatusCode != status {
						t.Fatalf("lost HTTP outcome: %v", err)
					}
				} else if err != nil || r == nil || r.Current != (status == 202) {
					t.Fatalf("invalid receipt: %v", err)
				}
				if calls != 1 {
					t.Fatal("implicit retry or mutation")
				}
			})
		}
	}
}

func TestForkLaunchRejectsInputBeforeSending(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			calls := 0
			c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { calls++; t.Error("invalid input sent") })
			defer server.Close()
			for _, change := range []func(*RootFSForkLaunchOptions){
				func(q *RootFSForkLaunchOptions) { q.WriterEpoch = 0 }, func(q *RootFSForkLaunchOptions) { q.WriterEpoch = 9007199254740992 },
				func(q *RootFSForkLaunchOptions) { q.ForkOperationID = "" }, func(q *RootFSForkLaunchOptions) { q.FilesystemID = "bad id" },
				func(q *RootFSForkLaunchOptions) { q.HeadGenerationID = strings.Repeat("a", 513) }, func(q *RootFSForkLaunchOptions) { q.IdempotencyKey = "" },
				func(q *RootFSForkLaunchOptions) { q.IdempotencyKey = "bad key" }, func(q *RootFSForkLaunchOptions) { q.ExpectedOperationID = "bad id" },
			} {
				q := forkLaunchOptions
				change(&q)
				if _, err := callForkLaunch(t, c, method, q); err == nil {
					t.Error("invalid input accepted")
				}
			}
			if calls != 0 {
				t.Fatal("invalid input reached transport")
			}
		})
	}
}

func TestForkLaunchRejectsForeignAndInvalidEvidence(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			data := forkLaunchFixture(false)
			calls := 0
			c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
			})
			defer server.Close()
			changes := []map[string]any{
				{"sandbox_id": "foreign"}, {"fork_operation_id": "foreign"}, {"filesystem_id": "foreign"}, {"head_generation_id": "foreign"},
				{"writer_epoch": 8}, {"operation_id": "different"}, {"lifecycle_epoch": 2}, {"runtime_generation": 1}, {"current": "true"},
				{"current": nil}, {"state": "unknown"}, {"state": "aborted"}, {"launch_digest": strings.Repeat("a", 64)},
				{"launch_digest": nil}, {"completed_at": nil}, {"completed_at": "2026-10-17T00:00:00Z"},
			}
			for _, change := range changes {
				data = forkLaunchFixture(false)
				for k, v := range change {
					data[k] = v
				}
				if _, err := callForkLaunch(t, c, method, forkLaunchOptions); err == nil {
					t.Errorf("accepted invalid pending evidence %v", change)
				}
			}
			for _, change := range []map[string]any{{"runtime_generation": 0}, {"launch_digest": nil}, {"launch_digest": "bad"}, {"completed_at": nil}, {"completed_at": 1792195200000}, {"completed_at": "2026-10-17"}, {"completed_at": "invalid"}} {
				data = forkLaunchFixture(true)
				for k, v := range change {
					data[k] = v
				}
				if _, err := callForkLaunch(t, c, method, forkLaunchOptions); err == nil {
					t.Errorf("accepted invalid completed evidence %v", change)
				}
			}
			data = forkLaunchFixture(false)
			data["state"] = "aborted"
			data["current"] = false
			if r, err := callForkLaunch(t, c, method, forkLaunchOptions); err != nil || r.State != "aborted" {
				t.Fatalf("aborted receipt rejected: %v", err)
			}
			data = forkLaunchFixture(false)
			data["current"] = false
			if r, err := callForkLaunch(t, c, method, forkLaunchOptions); err != nil || r.Current {
				t.Fatalf("old pending receipt rejected: %v", err)
			}
			if calls != len(changes)+9 {
				t.Fatalf("unexpected calls %d", calls)
			}
		})
	}
}

func TestForkLaunchLostReplyThenOriginalLookup(t *testing.T) {
	calls := 0
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			if r.Method != "POST" {
				t.Error("expected initial submission")
			}
			h := w.(http.Hijacker)
			conn, _, err := h.Hijack()
			if err != nil {
				t.Fatal(err)
			}
			_ = conn.Close()
			return
		}
		if r.Method != "GET" || r.URL.Query().Get("writer_epoch") != "7" {
			t.Error("uncertainty changed request")
		}
		writeJSON(t, w, 200, map[string]any{"success": true, "data": forkLaunchFixture(true)})
	})
	defer server.Close()
	if _, err := c.LaunchSandboxRootFSFork(t.Context(), "child", forkLaunchOptions); err == nil {
		t.Fatal("lost reply accepted")
	}
	if calls != 1 {
		t.Fatal("submission retried")
	}
	if r, err := c.GetSandboxRootFSForkLaunch(t.Context(), "child", forkLaunchOptions); err != nil || r.Current {
		t.Fatalf("retained history lost: %v", err)
	}
	if calls != 2 {
		t.Fatal("lookup submitted work")
	}
}

func TestForkLaunchAcceptedTerminalRejected(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 202, map[string]any{"success": true, "data": forkLaunchFixture(true)})
	})
	defer server.Close()
	if _, err := c.LaunchSandboxRootFSFork(t.Context(), "child", forkLaunchOptions); err == nil {
		t.Fatal("accepted terminal receipt")
	}
}
