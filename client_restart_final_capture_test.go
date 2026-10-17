package sandbox0

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

var captureTestID = strings.Repeat("a", 64)

func captureTestStatus() map[string]any {
	return map[string]any{"operation_id": restartTestID, "sandbox_id": "sandbox", "from_runtime_generation": 7, "state": "stopped", "capture_id": captureTestID, "stopped_at": "2026-10-15T00:00:00Z"}
}
func captureTestFile() map[string]any {
	return map[string]any{"operation_id": restartTestID, "capture_id": captureTestID, "runtime_generation": 7, "mutation_sequence": "18446744073709551615", "size_bytes": 8388608, "found": true, "content": "eA=="}
}

func TestRestartFinalCaptureExactIdentityAndReplay(t *testing.T) {
	prefix := "/api/v1/sandboxes/sandbox/restart-operations/" + restartTestID + "/final-capture"
	paths := []string{prefix, prefix + "/namespaces/native/auth.json", prefix + "/acknowledgement", prefix + "/acknowledgement"}
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls >= len(paths) || r.URL.Path != paths[calls] {
			t.Error("unexpected capture path")
		}
		calls++
		data := captureTestStatus()
		if calls == 2 {
			if r.Method != "GET" || r.URL.Query().Get("capture_id") != captureTestID {
				t.Error("capture identity lost")
			}
			data = captureTestFile()
		} else if calls >= 3 {
			var body map[string]string
			if r.Method != "POST" || json.NewDecoder(r.Body).Decode(&body) != nil || len(body) != 2 || body["capture_id"] != captureTestID || body["receipt_id"] != "receipt-1" {
				t.Error("ack identity changed")
			}
			data["state"] = "acknowledged"
			data["receipt_id"] = "receipt-1"
		} else if r.Method != "GET" {
			t.Error("metadata lookup mutates state")
		}
		writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
	})
	defer server.Close()
	if _, err := client.GetSandboxRestartFinalCapture(t.Context(), "sandbox", restartTestID); err != nil {
		t.Fatal(err)
	}
	file, err := client.ReadSandboxRestartFinalCaptureFile(t.Context(), "sandbox", restartTestID, captureTestID, "native", "auth.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(file.Content) != "x" || !file.Found {
		t.Fatal("content changed")
	}
	for i := 0; i < 2; i++ {
		if _, err := client.AcknowledgeSandboxRestartFinalCapture(t.Context(), "sandbox", restartTestID, captureTestID, "receipt-1"); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 4 {
		t.Fatal("unexpected automatic retry")
	}
}

func TestRestartFinalCaptureAbsenceAndErrors(t *testing.T) {
	for _, found := range []bool{false, true} {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			data := captureTestFile()
			data["found"] = found
			data["content"] = ""
			writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
		})
		result, err := client.ReadSandboxRestartFinalCaptureFile(t.Context(), "sandbox", restartTestID, captureTestID, "native", "auth.json")
		server.Close()
		if err != nil || result == nil || result.Found != found || len(result.Content) != 0 {
			t.Fatal("empty and absent files conflated", err)
		}
	}
	for _, status := range []int{404, 409, 503} {
		calls := 0
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]string{"code": "unavailable", "message": "unavailable"}})
		})
		result, err := client.ReadSandboxRestartFinalCaptureFile(t.Context(), "sandbox", restartTestID, captureTestID, "native", "auth.json")
		if err == nil || result != nil {
			t.Fatal("error became absence")
		}
		if _, err := client.AcknowledgeSandboxRestartFinalCapture(t.Context(), "sandbox", restartTestID, captureTestID, "receipt-1"); err == nil {
			t.Fatal("error became ack")
		}
		server.Close()
		if calls != 2 {
			t.Fatal("unexpected retries")
		}
	}
}

func TestRestartFinalCaptureInvalidAuthority(t *testing.T) {
	for _, change := range []map[string]any{{"capture_id": strings.Repeat("b", 64)}, {"operation_id": "foreign"}, {"runtime_generation": 0}, {"mutation_sequence": "18446744073709551616"}, {"size_bytes": 0}, {"found": false}, {"content": "private-invalid"}} {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			data := captureTestFile()
			for k, v := range change {
				data[k] = v
			}
			writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
		})
		result, err := client.ReadSandboxRestartFinalCaptureFile(t.Context(), "sandbox", restartTestID, captureTestID, "native", "auth.json")
		server.Close()
		if err == nil || result != nil {
			t.Fatal("invalid response accepted")
		}
		if strings.Contains(err.Error(), "private-invalid") {
			t.Fatal("error exposed content")
		}
	}
	for _, change := range []map[string]any{{"sandbox_id": "foreign"}, {"state": "pending"}, {"capture_id": "bad"}, {"state": "acknowledged"}, {"receipt_id": "premature"}} {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			data := captureTestStatus()
			for k, v := range change {
				data[k] = v
			}
			writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
		})
		result, err := client.GetSandboxRestartFinalCapture(t.Context(), "sandbox", restartTestID)
		server.Close()
		if err == nil || result != nil {
			t.Fatal("invalid status accepted")
		}
	}
}

func TestRestartFinalCaptureInvalidInput(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { t.Fatal("invalid input dispatched") })
	defer server.Close()
	for _, capture := range []string{"", strings.Repeat("A", 64)} {
		if _, err := client.ReadSandboxRestartFinalCaptureFile(t.Context(), "sandbox", restartTestID, capture, "native", "auth.json"); err == nil {
			t.Fatal("invalid capture accepted")
		}
	}
	for _, namespace := range []string{"..", "a/b", ".s0-write-secret", ""} {
		if _, err := client.ReadSandboxRestartFinalCaptureFile(t.Context(), "sandbox", restartTestID, captureTestID, namespace, "auth.json"); err == nil {
			t.Fatal("invalid namespace accepted")
		}
	}
	if _, err := client.GetSandboxRestartFinalCapture(t.Context(), "..", restartTestID); err == nil {
		t.Fatal("invalid sandbox accepted")
	}
	if _, err := client.AcknowledgeSandboxRestartFinalCapture(t.Context(), "sandbox", restartTestID, captureTestID, ""); err == nil {
		t.Fatal("invalid receipt accepted")
	}
}
