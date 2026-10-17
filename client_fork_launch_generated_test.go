package sandbox0

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestGeneratedForkLaunchPreservesOriginalConditions(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		for _, status := range []int{200, 202, 404, 409, 503} {
			if method == "GET" && status == 202 {
				continue
			}
			t.Run(fmt.Sprintf("%s-%d", method, status), func(t *testing.T) {
				calls := 0
				c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Method != method || r.URL.Path != "/api/v1/sandboxes/child/fork-launch" || r.Header.Get("Idempotency-Key") != "original" {
						t.Error("original request changed")
					}
					if method == "POST" {
						var body map[string]any
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Fatal(err)
						}
						if len(body) != 4 || body["writer_epoch"] != float64(7) || body["head_generation_id"] != "head" {
							t.Error("condition body changed")
						}
					} else {
						q := r.URL.Query()
						if len(q) != 4 || q.Get("fork_operation_id") != "fork" || q.Get("filesystem_id") != "child" || q.Get("head_generation_id") != "head" || q.Get("writer_epoch") != "7" {
							t.Error("lookup conditions changed")
						}
					}
					if status >= 400 {
						writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "retain request"}})
						return
					}
					writeJSON(t, w, status, map[string]any{"success": true, "data": map[string]any{"operation_id": "launch", "sandbox_id": "child", "fork_operation_id": "fork", "filesystem_id": "child", "head_generation_id": "head", "writer_epoch": 7, "lifecycle_epoch": 1, "runtime_generation": 0, "state": "pending", "current": true}})
				})
				defer server.Close()
				var result any
				var err error
				if method == "POST" {
					result, err = c.api.LaunchSandboxRootFSFork(t.Context(), &apispec.SandboxRootFSForkLaunchRequest{ForkOperationID: "fork", FilesystemID: "child", HeadGenerationID: "head", WriterEpoch: 7}, apispec.LaunchSandboxRootFSForkParams{ID: "child", IdempotencyKey: "original"})
				} else {
					result, err = c.api.GetSandboxRootFSForkLaunch(t.Context(), apispec.GetSandboxRootFSForkLaunchParams{ID: "child", IdempotencyKey: "original", ForkOperationID: "fork", FilesystemID: "child", HeadGenerationID: "head", WriterEpoch: 7})
				}
				if status < 400 {
					if err != nil || result == nil {
						t.Fatalf("generated response failed: %v", err)
					}
				} else {
					var apiErr *APIError
					if !errors.As(err, &apiErr) || apiErr.StatusCode != status {
						t.Fatalf("original HTTP outcome lost: %v", err)
					}
				}
				if calls != 1 {
					t.Fatal("implicit retry or mutation")
				}
			})
		}
	}
}
