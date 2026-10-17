package sandbox0

import (
	"encoding/json"
	"fmt"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"net/http"
	"testing"
)

func TestConditionalSnapshotRetainsExactRequestOnRetry(t *testing.T) {
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/api/v1/sandboxes/source/snapshots/capture" || r.Header.Get("Idempotency-Key") != "capture-7" {
			t.Error("unexpected request")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil || body["expected_runtime_generation"] != float64(7) {
			t.Error("generation changed")
		}
		if calls == 1 {
			writeJSON(t, w, 503, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "uncertain"}})
			return
		}
		writeJSON(t, w, 201, map[string]any{"success": true, "data": map[string]any{"id": "snapshot", "sandbox_id": "source", "generation_id": "captured-head", "created_at": "2026-10-16T00:00:00Z"}})
	})
	defer server.Close()
	req := &apispec.CaptureSandboxRootFSSnapshotRequest{ExpectedRuntimeGeneration: 7}
	if _, err := client.CaptureSandboxRootFSSnapshot(t.Context(), "source", req, "capture-7"); err == nil || calls != 1 {
		t.Fatal("uncertainty must not retry or fall back")
	}
	result, err := client.CaptureSandboxRootFSSnapshot(t.Context(), "source", req, "capture-7")
	if err != nil || result.ID != "snapshot" || !result.GenerationID.Set || result.GenerationID.Value != "captured-head" || calls != 2 {
		t.Fatalf("retry failed: %v", err)
	}
}

func TestConditionalSnapshotRejectsInvalidGenerationBeforeRequest(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected network request") })
	defer server.Close()
	for _, generation := range []int64{-1, 0, 9007199254740992} {
		if _, err := client.CaptureSandboxRootFSSnapshot(t.Context(), "source", &apispec.CaptureSandboxRootFSSnapshotRequest{ExpectedRuntimeGeneration: generation}, "capture"); err == nil {
			t.Fatal("invalid generation accepted")
		}
	}
}

func TestSnapshotObservationOnlyReadsOriginalCoordinates(t *testing.T) {
	for _, status := range []int{200, 404, 409, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/api/v1/sandboxes/source/snapshots/capture" || r.URL.Query().Get("expected_runtime_generation") != "7" || r.Header.Get("Idempotency-Key") != "capture-7" {
					t.Error("unexpected request")
				}
				if status == 200 {
					writeJSON(t, w, status, map[string]any{"success": true, "data": map[string]any{"id": "original", "sandbox_id": "source", "generation_id": "captured-head", "created_at": "2026-10-16T00:00:00Z"}})
				} else {
					writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "retained request"}})
				}
			})
			defer server.Close()
			got, err := client.GetCapturedSandboxRootFSSnapshot(t.Context(), "source", 7, "capture-7")
			if status == 200 {
				if err != nil || got == nil || got.ID != "original" || !got.GenerationID.Set || got.GenerationID.Value != "captured-head" {
					t.Fatalf("bad receipt: %v", err)
				}
			} else if err == nil {
				t.Fatal("missing outcome accepted")
			}
			if calls != 1 {
				t.Fatal("observation retried or dispatched")
			}
		})
	}
}
