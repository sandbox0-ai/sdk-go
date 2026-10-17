package sandbox0

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"testing"
)

const restartTestID = "40ea02de-5ef4-4710-a742-c9854cdfb6ce"

func restartTestReceipt() map[string]any {
	return map[string]any{"id": restartTestID, "sandbox_id": "sandbox", "request_key": "retry-key", "from_runtime_generation": 7, "state": "pausing", "created_at": "2026-10-15T00:00:00Z", "updated_at": "2026-10-15T00:00:00Z"}
}
func TestRestartExactRequestAndReadOnlyRecovery(t *testing.T) {
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		data := restartTestReceipt()
		status := 202
		if calls == 1 {
			if r.Method != "POST" || r.URL.Path != "/api/v1/sandboxes/sandbox/restart" || r.Header.Get("Idempotency-Key") != "retry-key" {
				t.Error("submission identity lost")
			}
			var body map[string]int64
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["expected_runtime_generation"] != 7 || len(body) != 1 {
				t.Error("generation changed")
			}
		} else {
			if r.Method != "GET" {
				t.Error("receipt lookup mutated sandbox")
			}
			status = 200
			data["state"] = "succeeded"
			data["result_runtime_generation"] = 8
			if calls == 2 && (r.URL.Path != "/api/v1/sandboxes/sandbox/restart-operations" || r.URL.Query().Get("request_key") != "retry-key") {
				t.Error("lookup key lost")
			}
			if calls == 3 && r.URL.Path != "/api/v1/sandboxes/sandbox/restart-operations/"+restartTestID {
				t.Error("operation ID lost")
			}
		}
		writeJSON(t, w, status, map[string]any{"success": true, "data": data})
	})
	defer server.Close()
	pending, err := client.RestartSandbox(t.Context(), "sandbox", 7, "retry-key")
	if err != nil {
		t.Fatal(err)
	}
	if pending.State != "pausing" || pending.ResultRuntimeGeneration.IsSet() {
		t.Fatal("acceptance reported completion")
	}
	found, err := client.FindSandboxRestartOperation(t.Context(), "sandbox", "retry-key")
	if err != nil {
		t.Fatal(err)
	}
	if generation, _ := found.ResultRuntimeGeneration.Get(); generation != 8 {
		t.Fatal("lost success generation")
	}
	if _, err = client.GetSandboxRestartOperation(t.Context(), "sandbox", restartTestID); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("unexpected retries: %d", calls)
	}
}
func TestRestartErrorNeverRetriesOrFallsBack(t *testing.T) {
	for _, status := range []int{404, 409, 503} {
		calls := 0
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]string{"code": "unavailable", "message": "unavailable"}})
		})
		_, err := client.RestartSandbox(t.Context(), "sandbox", 7, "retry-key")
		server.Close()
		if err == nil || calls != 1 {
			t.Fatalf("error or retry incorrect: %v, %d", err, calls)
		}
	}
}
func TestRestartRejectsInvalidInputsAndReceipts(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("invalid input reached transport") })
	defer server.Close()
	for _, generation := range []int64{-1, math.MaxInt64} {
		if _, err := client.RestartSandbox(t.Context(), "sandbox", generation, "retry-key"); err == nil {
			t.Fatal("invalid generation accepted")
		}
	}
	for _, key := range []string{"", " spaced", strings.Repeat("x", 256)} {
		if _, err := client.RestartSandbox(t.Context(), "sandbox", 7, key); err == nil {
			t.Fatal("invalid key accepted")
		}
	}
	for _, change := range []map[string]any{{"sandbox_id": "foreign"}, {"request_key": "wrong"}, {"from_runtime_generation": 8}, {"state": "succeeded"}, {"state": "succeeded", "result_runtime_generation": 9}, {"result_runtime_generation": 8}, {"state": "unknown"}} {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			data := restartTestReceipt()
			for k, v := range change {
				data[k] = v
			}
			writeJSON(t, w, 202, map[string]any{"success": true, "data": data})
		})
		_, err := client.RestartSandbox(t.Context(), "sandbox", 7, "retry-key")
		server.Close()
		if err == nil {
			t.Fatalf("invalid receipt accepted: %v", change)
		}
	}
}
func TestRestartExactInt64AndZero(t *testing.T) {
	for _, generation := range []int64{0, math.MaxInt64 - 1} {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			var body map[string]int64
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["expected_runtime_generation"] != generation {
				t.Fatal("generation precision lost")
			}
			data := restartTestReceipt()
			data["from_runtime_generation"] = generation
			data["state"] = "succeeded"
			data["result_runtime_generation"] = generation + 1
			writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
		})
		_, err := client.RestartSandbox(t.Context(), "sandbox", generation, "retry-key")
		server.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestRestartStorageDiagnosticDecodesWithoutCompletion(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		data := restartTestReceipt()
		data["state"] = "resuming"
		data["retry"] = map[string]any{"attempts": 3, "observed_stage": "resuming", "blocked_reason": "storage_not_writable"}
		writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
	})
	defer server.Close()
	result, err := client.GetSandboxRestartOperation(t.Context(), "sandbox", restartTestID)
	if err != nil {
		t.Fatal(err)
	}
	retry, ok := result.Retry.Get()
	if !ok {
		t.Fatal("retry absent")
	}
	reason, ok := retry.BlockedReason.Get()
	if !ok || reason != "storage_not_writable" || result.ResultRuntimeGeneration.IsSet() {
		t.Fatal("diagnostic or completion changed")
	}
}
