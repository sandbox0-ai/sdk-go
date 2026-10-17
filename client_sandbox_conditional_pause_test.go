package sandbox0

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestConditionalMemoryPauseRetainsIntentAfterUncertainReply(t *testing.T) {
	calls := 0
	operation := "idem-" + strings.Repeat("a", 64)
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/sandboxes/sb_1/pause" || r.Header.Get("Idempotency-Key") != "pause-7" {
			t.Fatalf("wrong pause route/key")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 2 || body["memory"] != true || body["expected_runtime_generation"] != float64(7) {
			t.Fatalf("wrong body: %v", body)
		}
		if calls == 1 {
			writeJSON(t, w, 503, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "reply uncertain"}})
			return
		}
		status := 202
		if calls == 3 {
			status = 200
		}
		writeJSON(t, w, status, map[string]any{"success": true, "data": map[string]any{"sandbox_id": "sb_1", "paused": calls == 3, "operation_id": operation, "runtime_generation": 7}})
	})
	defer server.Close()
	if _, err := client.PauseSandboxMemoryForOperation(t.Context(), "sb_1", 7, "pause-7"); err == nil || calls != 1 {
		t.Fatalf("uncertain response: calls=%d err=%v", calls, err)
	}
	for _, paused := range []bool{false, true} {
		result, err := client.PauseSandboxMemoryForOperation(t.Context(), "sb_1", 7, "pause-7")
		if err != nil {
			t.Fatal(err)
		}
		if result.Paused != paused || result.OperationID.Value != operation || result.RuntimeGeneration.Value != 7 {
			t.Fatalf("wrong result: %+v", result)
		}
	}
	if calls != 3 {
		t.Fatalf("requests=%d", calls)
	}
}

func TestConditionalMemoryPauseRejectsInvalidInputAndUnboundResponses(t *testing.T) {
	calls := 0
	data := map[string]any{"sandbox_id": "sb_1", "paused": true, "operation_id": "idem-" + strings.Repeat("a", 64), "runtime_generation": 7}
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
	})
	defer server.Close()
	for _, generation := range []int64{0, -1, 9007199254740992} {
		if _, err := client.PauseSandboxMemoryForOperation(t.Context(), "sb_1", generation, "pause-7"); err == nil {
			t.Fatal("invalid generation accepted")
		}
	}
	for _, key := range []string{"", " pause-7", "pause/7", strings.Repeat("x", 256)} {
		if _, err := client.PauseSandboxMemoryForOperation(t.Context(), "sb_1", 7, key); err == nil {
			t.Fatal("invalid key accepted")
		}
	}
	if calls != 0 {
		t.Fatalf("invalid inputs sent %d requests", calls)
	}
	for _, tc := range []struct {
		field string
		value any
	}{
		{"operation_id", nil}, {"operation_id", "other"}, {"runtime_generation", nil}, {"runtime_generation", 8}, {"sandbox_id", "other"}, {"paused", nil},
	} {
		original := data[tc.field]
		if tc.value == nil {
			delete(data, tc.field)
		} else {
			data[tc.field] = tc.value
		}
		if _, err := client.PauseSandboxMemoryForOperation(t.Context(), "sb_1", 7, "pause-7"); err == nil {
			t.Fatalf("unbound response accepted: %s=%v", tc.field, tc.value)
		}
		data[tc.field] = original
	}
	if calls != 6 {
		t.Fatalf("response failure caused implicit requests: %d", calls)
	}
}
