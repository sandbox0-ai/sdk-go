package sandbox0

import (
	"net/http"
	"strings"
	"testing"
)

func memoryRestorationStep(from, to int64) map[string]any {
	return map[string]any{"operation_id": "restore", "capture_operation_id": "capture", "from_runtime_generation": from, "to_runtime_generation": to, "restore_id": strings.Repeat("a", 64), "managed_files_size_bytes": int64(8388608)}
}
func memoryRestorationData() map[string]any {
	return map[string]any{"sandbox_id": "sandbox", "runtime_generation": int64(5), "filesystem_id": "filesystem", "runtime_compatibility_digest": "sha256:" + strings.Repeat("b", 64), "steps": []map[string]any{memoryRestorationStep(1, 3), memoryRestorationStep(3, 5)}, "observed_at": "2026-10-15T00:00:00Z"}
}
func TestMemoryRestorationExactRead(t *testing.T) {
	for _, size := range []int64{0, 8388608} {
		calls := 0
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if r.Method != "GET" || r.URL.Path != "/api/v1/sandboxes/sandbox/memory-restoration" || r.URL.RawQuery != "runtime_generation=5" {
				t.Error("unexpected request")
			}
			value := memoryRestorationData()
			for _, s := range value["steps"].([]map[string]any) {
				s["managed_files_size_bytes"] = size
			}
			writeJSON(t, w, 200, map[string]any{"success": true, "data": value})
		})
		result, err := client.GetSandboxMemoryRestoration(t.Context(), "sandbox", 5)
		server.Close()
		if err != nil {
			t.Fatal(err)
		}
		if calls != 1 || len(result.Steps) != 2 || result.Steps[0].FromRuntimeGeneration != 1 || result.Steps[0].ToRuntimeGeneration != 3 || result.Steps[1].ToRuntimeGeneration != 5 || result.Steps[0].ManagedFilesSizeBytes != size {
			t.Fatal("lineage changed")
		}
	}
}
func TestMemoryRestorationRejectsMalformedAuthority(t *testing.T) {
	changes := []map[string]any{{"sandbox_id": "foreign"}, {"runtime_generation": 4}, {"filesystem_id": ""}, {"runtime_compatibility_digest": "bad"}, {"steps": []any{}}, {"steps": []any{memoryRestorationStep(1, 3), memoryRestorationStep(4, 5)}}, {"steps": []any{memoryRestorationStep(1, 3)}}, {"steps": []any{memoryRestorationStep(5, 5)}}}
	for _, change := range []map[string]any{{"restore_id": "bad"}, {"operation_id": ""}, {"capture_operation_id": ""}, {"from_runtime_generation": true}, {"managed_files_size_bytes": -1}, {"managed_files_size_bytes": 67108865}} {
		step := memoryRestorationStep(1, 5)
		for key, value := range change {
			step[key] = value
		}
		changes = append(changes, map[string]any{"steps": []any{step}})
	}
	for _, change := range changes {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			value := memoryRestorationData()
			for key, v := range change {
				value[key] = v
			}
			writeJSON(t, w, 200, map[string]any{"success": true, "data": value})
		})
		_, err := client.GetSandboxMemoryRestoration(t.Context(), "sandbox", 5)
		server.Close()
		if err == nil {
			t.Fatalf("accepted malformed authority: %v", change)
		}
	}
}
func TestMemoryRestorationNoRetryOrFallback(t *testing.T) {
	for _, status := range []int{404, 409, 503} {
		calls := 0
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "unavailable"}})
		})
		_, err := client.GetSandboxMemoryRestoration(t.Context(), "sandbox", 5)
		server.Close()
		if err == nil || calls != 1 {
			t.Fatal("unexpected success or retry")
		}
	}
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("invalid input reached transport") })
	defer server.Close()
	for _, generation := range []int64{0, -1} {
		if _, err := client.GetSandboxMemoryRestoration(t.Context(), "sandbox", generation); err == nil {
			t.Fatal("invalid generation accepted")
		}
	}
	for _, sandbox := range []string{"", "..", "a/b", "a?b"} {
		if _, err := client.GetSandboxMemoryRestoration(t.Context(), sandbox, 5); err == nil {
			t.Fatal("invalid sandbox accepted")
		}
	}
}
