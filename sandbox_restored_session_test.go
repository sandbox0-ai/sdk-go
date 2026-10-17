package sandbox0

import (
	"net/http"
	"testing"
)

func TestRestoredLookupPreservesOriginalAttemptAndNoFallback(t *testing.T) {
	calls, status, origin := 0, 200, 1
	phase := "running"
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/api/v1/sandboxes/sandbox/sessions/conditional/lookup" || r.URL.Query().Get("runtime_generation") != "5" || r.URL.Query().Get("creation_runtime_generation") != "1" {
			t.Error("incorrect restored lookup")
		}
		if status != 200 {
			writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "unavailable"}})
			return
		}
		data := executionSessionFixture()
		data["runtime_generation"] = 5
		data["phase"] = phase
		data["attempt"] = map[string]any{"id": "original", "number": 1, "runtime_generation": origin}
		writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
	})
	defer server.Close()
	sb := client.Sandbox("sandbox")
	value, err := sb.FindRestoredConditionalSession(t.Context(), "original-key", 5, 1)
	if err != nil || value.Attempt.Value.ID != "original" {
		t.Fatalf("lookup: %v", err)
	}
	origin = 5
	if _, err = sb.FindRestoredConditionalSession(t.Context(), "original-key", 5, 1); err == nil {
		t.Fatal("replacement adopted")
	}
	origin = 1
	phase = "stopped"
	if _, err = sb.FindRestoredConditionalSession(t.Context(), "original-key", 5, 1); err == nil {
		t.Fatal("stopped process adopted")
	}
	for _, code := range []int{404, 405, 409, 503} {
		status = code
		if _, err = sb.FindRestoredConditionalSession(t.Context(), "original-key", 5, 1); err == nil {
			t.Fatal("error adopted")
		}
	}
	for _, invalid := range []int64{0, -1, 5, 6} {
		if _, err = sb.FindRestoredConditionalSession(t.Context(), "original-key", 5, invalid); err == nil {
			t.Fatal("invalid origin adopted")
		}
	}
	if calls != 7 {
		t.Fatalf("unexpected I/O count %d", calls)
	}
}
