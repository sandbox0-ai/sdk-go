package sandbox0

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func snapshotRestoreTestRequest() apispec.SandboxSnapshotRestoreRequest {
	return apispec.SandboxSnapshotRestoreRequest{ExpectedRuntimeGeneration: 7, SnapshotID: "snapshot", SnapshotGenerationID: "generation", DiscardProcessState: true}
}
func snapshotRestoreTestReceipt() map[string]any {
	r := restartTestReceipt()
	r["snapshot_id"] = "snapshot"
	r["snapshot_generation_id"] = "generation"
	r["discard_process_state"] = true
	return r
}
func TestSnapshotRestoreExactTransportAndReadOnlyRecovery(t *testing.T) {
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		data := snapshotRestoreTestReceipt()
		status := 202
		if calls == 1 {
			if r.Method != "POST" || r.URL.Path != "/api/v1/sandboxes/sandbox/snapshot-restore" || r.Header.Get("Idempotency-Key") != "retry-key" {
				t.Error("restore identity changed")
			}
			var body map[string]any
			if json.NewDecoder(r.Body).Decode(&body) != nil || len(body) != 4 || body["snapshot_id"] != "snapshot" || body["snapshot_generation_id"] != "generation" || body["expected_runtime_generation"] != float64(7) || body["discard_process_state"] != true {
				t.Error("restore request changed")
			}
		} else {
			if r.Method != "GET" {
				t.Error("observation submitted a mutation")
			}
			if calls == 2 && (r.URL.Path != "/api/v1/sandboxes/sandbox/snapshot-restore-operations" || r.URL.Query().Get("request_key") != "retry-key") {
				t.Error("lookup key changed")
			}
			if calls == 3 && r.URL.Path != "/api/v1/sandboxes/sandbox/snapshot-restore-operations/"+restartTestID {
				t.Error("operation ID changed")
			}
			data["state"] = "succeeded"
			data["result_runtime_generation"] = 8
			data["restored_rootfs_generation_id"] = "restored"
			status = 200
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
	})
	defer server.Close()
	if _, err := client.RestoreSandboxSnapshot(t.Context(), "sandbox", snapshotRestoreTestRequest(), "retry-key"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.FindSandboxSnapshotRestoreOperation(t.Context(), "sandbox", "retry-key"); err != nil {
		t.Fatal(err)
	}
	result, err := client.GetSandboxSnapshotRestoreOperation(t.Context(), "sandbox", restartTestID)
	if err != nil || result.ResultRuntimeGeneration.Value != 8 || result.RestoredRootfsGenerationID.Value != "restored" {
		t.Fatalf("invalid receipt: %v", err)
	}
}
func TestSnapshotRestoreRejectsMismatchesAndNeverFallsBack(t *testing.T) {
	for _, change := range []map[string]any{{"snapshot_id": "other"}, {"snapshot_generation_id": "other"}, {"discard_process_state": false}, {"state": "succeeded", "result_runtime_generation": 8}, {"state": "resuming"}, {"result_runtime_generation": 8}} {
		calls := 0
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			data := snapshotRestoreTestReceipt()
			for k, v := range change {
				data[k] = v
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
		})
		_, err := client.RestoreSandboxSnapshot(t.Context(), "sandbox", snapshotRestoreTestRequest(), "retry-key")
		server.Close()
		if err == nil || calls != 1 {
			t.Fatalf("mismatch accepted or retried: %v", change)
		}
	}
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("invalid input reached transport") })
	defer server.Close()
	for _, change := range []func(*apispec.SandboxSnapshotRestoreRequest){func(q *apispec.SandboxSnapshotRestoreRequest) { q.DiscardProcessState = false }, func(q *apispec.SandboxSnapshotRestoreRequest) { q.ExpectedRuntimeGeneration = 0 }, func(q *apispec.SandboxSnapshotRestoreRequest) { q.SnapshotID = "" }, func(q *apispec.SandboxSnapshotRestoreRequest) { q.ExpectedRuntimeGeneration = 9007199254740991 }} {
		q := snapshotRestoreTestRequest()
		change(&q)
		if _, err := client.RestoreSandboxSnapshot(t.Context(), "sandbox", q, "retry-key"); err == nil {
			t.Error("invalid input accepted")
		}
	}
}
