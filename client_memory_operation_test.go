package sandbox0

import (
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"net/http"
	"strings"
	"testing"
)

func TestMemoryOperationExactReadAndTerminalStates(t *testing.T) {
	operation := "idem-" + strings.Repeat("b", 64)
	for _, action := range []apispec.LookupSandboxMemoryOperationAction{"pause", "resume"} {
		for _, state := range []string{"pending", "completed", "canceled", "failed"} {
			for _, current := range []bool{false, true} {
				if current && (state == "pending" || (action == "pause" && state == "failed")) {
					continue
				}
				input := MemoryOperationLookupOptions{Action: action, ExpectedRuntimeGeneration: 7, IdempotencyKey: "pause-7", ExpectedOperationID: operation}
				data := map[string]any{"sandbox_id": "sb_1", "action": string(action), "operation_id": operation, "checkpoint_id": operation,
					"from_runtime_generation": 7, "lifecycle_epoch": 3, "state": state, "current": current}
				if action == "resume" {
					input.CheckpointID = "capture-7"
					data["checkpoint_id"] = input.CheckpointID
				}
				if state == "canceled" {
					if action == "resume" {
						data["reason"] = "before_restore"
					} else {
						data["reason"] = "before_capture"
					}
				}
				if state == "failed" {
					if action == "resume" {
						data["reason"] = "restore_failed"
					} else {
						data["reason"] = "capture_failed"
					}
				}
				if action == "resume" && (state == "completed" || state == "failed") {
					data["result_runtime_generation"] = 8
				}
				calls := 0
				client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					calls++
					query := r.URL.Query()
					count := 2
					if action == "resume" {
						count = 3
					}
					if r.Method != "GET" || r.URL.Path != "/api/v1/sandboxes/sb_1/memory-operations/lookup" || r.Header.Get("Idempotency-Key") != input.IdempotencyKey ||
						len(query) != count || query.Get("action") != string(action) || query.Get("expected_runtime_generation") != "7" || query.Get("checkpoint_id") != input.CheckpointID {
						t.Errorf("lookup changed original coordinates")
					}
					writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
				})
				result, err := client.LookupSandboxMemoryOperation(t.Context(), "sb_1", input)
				if err != nil {
					t.Fatal(err)
				}
				if string(result.State) != state || result.Current != current || calls != 1 {
					t.Fatal("wrong observation or implicit retry")
				}
				server.Close()
			}
		}
	}
}

func TestMemoryOperationRejectsUnboundEvidenceAndRetainsErrors(t *testing.T) {
	operation := "idem-" + strings.Repeat("b", 64)
	input := MemoryOperationLookupOptions{Action: "pause", ExpectedRuntimeGeneration: 7, IdempotencyKey: "pause-7", ExpectedOperationID: operation}
	data := map[string]any{"sandbox_id": "sb_1", "action": "pause", "operation_id": operation, "checkpoint_id": operation,
		"from_runtime_generation": 7, "lifecycle_epoch": 3, "state": "canceled", "reason": "before_capture", "current": true}
	status, calls := 200, 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if status != 200 {
			writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "no proof"}})
			return
		}
		writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
	})
	defer server.Close()
	for _, change := range []struct {
		field string
		value any
	}{
		{"sandbox_id", "other"}, {"action", "resume"}, {"operation_id", "idem-" + strings.Repeat("c", 64)}, {"checkpoint_id", "other"},
		{"from_runtime_generation", 8}, {"lifecycle_epoch", 0}, {"lifecycle_epoch", 1.5}, {"current", "true"}, {"current", nil},
		{"reason", "before_restore"}, {"reason", nil}, {"result_runtime_generation", 8}, {"state", "pending"}, {"state", "completed"}, {"state", "failed"},
	} {
		original, exists := data[change.field]
		if change.value == nil {
			delete(data, change.field)
		} else {
			data[change.field] = change.value
		}
		if _, err := client.LookupSandboxMemoryOperation(t.Context(), "sb_1", input); err == nil {
			t.Fatalf("unbound evidence accepted: %s", change.field)
		}
		if exists {
			data[change.field] = original
		} else {
			delete(data, change.field)
		}
	}
	before := calls
	for _, invalid := range []MemoryOperationLookupOptions{
		{Action: "unknown", ExpectedRuntimeGeneration: 7, IdempotencyKey: "key"},
		{Action: "pause", ExpectedRuntimeGeneration: 0, IdempotencyKey: "key"},
		{Action: "pause", ExpectedRuntimeGeneration: 7, IdempotencyKey: "bad/key"},
		{Action: "pause", ExpectedRuntimeGeneration: 7, IdempotencyKey: "key", CheckpointID: "forbidden"},
		{Action: "resume", ExpectedRuntimeGeneration: 7, IdempotencyKey: "key"},
		{Action: "resume", ExpectedRuntimeGeneration: 9007199254740991, IdempotencyKey: "key", CheckpointID: "capture"},
		{Action: "pause", ExpectedRuntimeGeneration: 7, IdempotencyKey: "key", ExpectedOperationID: "bad"},
	} {
		if _, err := client.LookupSandboxMemoryOperation(t.Context(), "sb_1", invalid); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
	if calls != before {
		t.Fatal("invalid input reached network")
	}
	for _, code := range []int{404, 409, 503} {
		status = code
		if _, err := client.LookupSandboxMemoryOperation(t.Context(), "sb_1", input); err == nil {
			t.Fatal("error became cancellation")
		}
	}
	if calls != before+3 {
		t.Fatal("implicit retries")
	}
}
