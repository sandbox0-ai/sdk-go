package sandbox0

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

var snapshotCaptureTestID = strings.Repeat("a", 64)

func snapshotCaptureTestStatus() map[string]any {
	return map[string]any{"operation_id": restartTestID, "sandbox_id": "sandbox", "from_runtime_generation": 7, "state": "stopped", "capture_id": snapshotCaptureTestID, "stopped_at": "2026-10-15T00:00:00Z"}
}
func snapshotCaptureTestFile() map[string]any {
	return map[string]any{"operation_id": restartTestID, "capture_id": snapshotCaptureTestID, "runtime_generation": 7, "mutation_sequence": "18446744073709551615", "size_bytes": 8388608, "found": true, "content": "eA=="}
}

func TestSnapshotRestoreFinalCaptureExactIdentityAndReplay(t *testing.T) {
	prefix := "/api/v1/sandboxes/sandbox/snapshot-restore-operations/" + restartTestID + "/final-capture"
	paths := []string{prefix, prefix + "/namespaces/native/auth.json", prefix + "/acknowledgement", prefix + "/acknowledgement"}
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls >= len(paths) || r.URL.Path != paths[calls] {
			t.Error("unexpected capture path")
		}
		calls++
		data := snapshotCaptureTestStatus()
		if calls == 2 {
			if r.Method != "GET" || r.URL.Query().Get("capture_id") != snapshotCaptureTestID {
				t.Error("capture identity lost")
			}
			data = snapshotCaptureTestFile()
		} else if calls >= 3 {
			var body map[string]string
			if r.Method != "POST" || json.NewDecoder(r.Body).Decode(&body) != nil || len(body) != 2 || body["capture_id"] != snapshotCaptureTestID || body["receipt_id"] != "receipt-1" {
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
	if _, err := client.GetSandboxSnapshotRestoreFinalCapture(t.Context(), "sandbox", restartTestID); err != nil {
		t.Fatal(err)
	}
	file, err := client.ReadSandboxSnapshotRestoreFinalCaptureFile(t.Context(), "sandbox", restartTestID, snapshotCaptureTestID, "native", "auth.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(file.Content) != "x" || !file.Found {
		t.Fatal("content changed")
	}
	for i := 0; i < 2; i++ {
		if _, err := client.AcknowledgeSandboxSnapshotRestoreFinalCapture(t.Context(), "sandbox", restartTestID, snapshotCaptureTestID, "receipt-1"); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 4 {
		t.Fatal("unexpected automatic retry")
	}
}

func TestSnapshotRestoreFinalCaptureAbsenceAndErrors(t *testing.T) {
	for _, found := range []bool{false, true} {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			data := snapshotCaptureTestFile()
			data["found"] = found
			data["content"] = ""
			writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
		})
		result, err := client.ReadSandboxSnapshotRestoreFinalCaptureFile(t.Context(), "sandbox", restartTestID, snapshotCaptureTestID, "native", "auth.json")
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
		result, err := client.ReadSandboxSnapshotRestoreFinalCaptureFile(t.Context(), "sandbox", restartTestID, snapshotCaptureTestID, "native", "auth.json")
		if err == nil || result != nil {
			t.Fatal("error became absence")
		}
		if _, err := client.AcknowledgeSandboxSnapshotRestoreFinalCapture(t.Context(), "sandbox", restartTestID, snapshotCaptureTestID, "receipt-1"); err == nil {
			t.Fatal("error became ack")
		}
		server.Close()
		if calls != 2 {
			t.Fatal("unexpected retries")
		}
	}
}

func TestSnapshotRestoreFinalCaptureInvalidAuthority(t *testing.T) {
	for _, change := range []map[string]any{{"capture_id": strings.Repeat("b", 64)}, {"operation_id": "foreign"}, {"runtime_generation": 0}, {"mutation_sequence": "18446744073709551616"}, {"size_bytes": 0}, {"found": false}, {"content": "private-invalid"}} {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			data := snapshotCaptureTestFile()
			for k, v := range change {
				data[k] = v
			}
			writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
		})
		result, err := client.ReadSandboxSnapshotRestoreFinalCaptureFile(t.Context(), "sandbox", restartTestID, snapshotCaptureTestID, "native", "auth.json")
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
			data := snapshotCaptureTestStatus()
			for k, v := range change {
				data[k] = v
			}
			writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
		})
		result, err := client.GetSandboxSnapshotRestoreFinalCapture(t.Context(), "sandbox", restartTestID)
		server.Close()
		if err == nil || result != nil {
			t.Fatal("invalid status accepted")
		}
	}
}

func TestSnapshotRestoreFinalCaptureInvalidInput(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { t.Fatal("invalid input dispatched") })
	defer server.Close()
	for _, capture := range []string{"", strings.Repeat("A", 64)} {
		if _, err := client.ReadSandboxSnapshotRestoreFinalCaptureFile(t.Context(), "sandbox", restartTestID, capture, "native", "auth.json"); err == nil {
			t.Fatal("invalid capture accepted")
		}
	}
	for _, namespace := range []string{"..", "a/b", ".s0-write-secret", ""} {
		if _, err := client.ReadSandboxSnapshotRestoreFinalCaptureFile(t.Context(), "sandbox", restartTestID, snapshotCaptureTestID, namespace, "auth.json"); err == nil {
			t.Fatal("invalid namespace accepted")
		}
	}
	if _, err := client.GetSandboxSnapshotRestoreFinalCapture(t.Context(), "..", restartTestID); err == nil {
		t.Fatal("invalid sandbox accepted")
	}
	if _, err := client.AcknowledgeSandboxSnapshotRestoreFinalCapture(t.Context(), "sandbox", restartTestID, snapshotCaptureTestID, ""); err == nil {
		t.Fatal("invalid receipt accepted")
	}
}
