package sandbox0

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

const forkObservationFixture = `{"operation_id":"fork-op","source_sandbox_id":"source","target_sandbox_id":"child","source_runtime_generation":7,"source_filesystem_id":"source-fs","source_head_generation_id":"source-head","target_filesystem_id":"child","target_head_generation_id":"child-head","base_artifact_digest":"base","source_oci_digest":"oci","writer_epoch":1,"managed_files_size_bytes":100,"auto_resume":false,"automatic_routes":false,"sandbox_created_at":"2026-10-17T00:00:00Z","filesystem_created_at":"2026-10-17T00:00:00Z","observed_at":"2026-10-17T00:00:00Z"}`

func TestForkObservationOnlyReadsOriginalCoordinates(t *testing.T) {
	for _, generation := range []int64{0, 7} {
		for _, status := range []int{200, 404, 409, 503} {
			t.Run(fmt.Sprintf("%d-%d", generation, status), func(t *testing.T) {
				calls := 0
				client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Method != "GET" || r.URL.Path != "/api/v1/sandboxes/source/fork" || r.URL.Query().Get("expected_runtime_generation") != strconv.FormatInt(generation, 10) || r.Header.Get("Idempotency-Key") != "original-fork" || r.ContentLength > 0 {
						t.Error("unexpected lookup request")
					}
					var data map[string]any
					if err := json.Unmarshal([]byte(forkObservationFixture), &data); err != nil {
						t.Fatal(err)
					}
					data["source_runtime_generation"] = generation
					if status == 200 {
						writeJSON(t, w, status, map[string]any{"success": true, "data": data})
					} else {
						writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "retain request"}})
					}
				})
				defer server.Close()
				got, err := client.GetSandboxRootFSFork(t.Context(), "source", generation, "original-fork")
				if status == 200 {
					if err != nil || got == nil || got.TargetSandboxID != "child" {
						t.Fatalf("bad receipt: %v", err)
					}
				} else if err == nil {
					t.Fatal("missing outcome accepted")
				}
				if calls != 1 {
					t.Fatal("lookup retried or submitted work")
				}
			})
		}
	}
}
func TestForkObservationRejectsInputAndMismatchedReceipt(t *testing.T) {
	for _, field := range []string{"source_sandbox_id", "source_runtime_generation", "target_filesystem_id", "writer_epoch", "auto_resume"} {
		t.Run(field, func(t *testing.T) {
			calls := 0
			client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				var data map[string]any
				if err := json.Unmarshal([]byte(forkObservationFixture), &data); err != nil {
					t.Fatal(err)
				}
				switch field {
				case "source_runtime_generation":
					data[field] = 8
				case "writer_epoch":
					data[field] = 9007199254740992
				case "auto_resume":
					delete(data, field)
				default:
					data[field] = "foreign"
				}
				writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
			})
			defer server.Close()
			for _, g := range []int64{-1, 9007199254740992} {
				if _, err := client.GetSandboxRootFSFork(t.Context(), "source", g, "original-fork"); err == nil {
					t.Fatal("invalid input accepted")
				}
			}
			if _, err := client.GetSandboxRootFSFork(t.Context(), "source", 7, ""); err == nil {
				t.Fatal("missing key accepted")
			}
			if calls != 0 {
				t.Fatal("invalid input sent")
			}
			if _, err := client.GetSandboxRootFSFork(t.Context(), "source", 7, "original-fork"); err == nil {
				t.Fatal("invalid receipt accepted")
			}
			if calls != 1 {
				t.Fatal("unexpected retries")
			}
		})
	}
}
