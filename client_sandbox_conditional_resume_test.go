package sandbox0

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestConditionalMemoryResumeRetainsExactIntentOnRetry(t *testing.T) {
	for _, generation := range []int64{0, 7} {
		calls := 0
		operation := "idem-" + strings.Repeat("b", 64)
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method != "POST" || r.URL.Path != "/api/v1/sandboxes/sb_1/resume" || r.Header.Get("Idempotency-Key") != "resume-7" {
				t.Fatal("wrong conditional resume route/key")
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body) != 3 || body["memory"] != true || body["expected_runtime_generation"] != float64(generation) || body["checkpoint_id"] != "capture-7" {
				t.Fatalf("wrong body: %v", body)
			}
			if calls == 1 {
				writeJSON(t, w, 503, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "restore reply uncertain"}})
				return
			}
			writeJSON(t, w, 200, map[string]any{"success": true, "data": map[string]any{"sandbox_id": "sb_1", "resumed": true,
				"operation_id": operation, "checkpoint_id": "capture-7", "from_runtime_generation": generation, "runtime_generation": generation + 1}})
		})
		if _, err := client.ResumeSandboxMemoryForOperation(t.Context(), "sb_1", generation, "capture-7", "resume-7"); err == nil || calls != 1 {
			t.Fatalf("uncertain result: calls=%d err=%v", calls, err)
		}
		for range 2 {
			result, err := client.ResumeSandboxMemoryForOperation(t.Context(), "sb_1", generation, "capture-7", "resume-7")
			if err != nil {
				t.Fatal(err)
			}
			if !result.Resumed || result.OperationID.Value != operation || result.CheckpointID.Value != "capture-7" ||
				result.FromRuntimeGeneration.Value != generation || result.RuntimeGeneration.Value != generation+1 {
				t.Fatalf("wrong result: %+v", result)
			}
		}
		if calls != 3 {
			t.Fatalf("requests=%d", calls)
		}
		server.Close()
	}
}

func TestConditionalMemoryResumeRejectsInvalidInputsAndUnboundResults(t *testing.T) {
	calls := 0
	data := map[string]any{"sandbox_id": "sb_1", "resumed": true, "operation_id": "idem-" + strings.Repeat("b", 64), "checkpoint_id": "capture-7", "from_runtime_generation": 7, "runtime_generation": 8}
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
	})
	defer server.Close()
	for _, generation := range []int64{-1, 9007199254740991} {
		if _, err := client.ResumeSandboxMemoryForOperation(t.Context(), "sb_1", generation, "capture-7", "resume-7"); err == nil {
			t.Fatal("invalid generation accepted")
		}
	}
	for _, checkpoint := range []string{"", " padded", "line\nbreak", "🙂", strings.Repeat("x", 257)} {
		if _, err := client.ResumeSandboxMemoryForOperation(t.Context(), "sb_1", 7, checkpoint, "resume-7"); err == nil {
			t.Fatal("invalid checkpoint accepted")
		}
	}
	for _, key := range []string{"", " padded", "resume/7", strings.Repeat("x", 256)} {
		if _, err := client.ResumeSandboxMemoryForOperation(t.Context(), "sb_1", 7, "capture-7", key); err == nil {
			t.Fatal("invalid key accepted")
		}
	}
	if calls != 0 {
		t.Fatalf("invalid input sent %d requests", calls)
	}
	for _, tc := range []struct {
		field string
		value any
	}{
		{"operation_id", nil}, {"operation_id", "other"}, {"checkpoint_id", nil}, {"checkpoint_id", "other"},
		{"from_runtime_generation", nil}, {"from_runtime_generation", 8}, {"runtime_generation", nil}, {"runtime_generation", 9},
		{"sandbox_id", "other"}, {"resumed", false}, {"resumed", nil},
	} {
		original := data[tc.field]
		if tc.value == nil {
			delete(data, tc.field)
		} else {
			data[tc.field] = tc.value
		}
		if _, err := client.ResumeSandboxMemoryForOperation(t.Context(), "sb_1", 7, "capture-7", "resume-7"); err == nil {
			t.Fatalf("unbound response accepted: %s=%v", tc.field, tc.value)
		}
		data[tc.field] = original
	}
	if calls != 11 {
		t.Fatalf("response failure caused implicit requests: %d", calls)
	}
}
