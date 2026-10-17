package sandbox0

import (
	"net/http"
	"testing"
)

func healthTestData() map[string]any {
	check := func(state, reason, source string) map[string]any {
		var at any = "2026-10-15T00:00:01Z"
		if source == "none" {
			at = nil
		}
		return map[string]any{"state": state, "reason": reason, "source": source, "observed_at": at}
	}
	return map[string]any{
		"sandbox_id": "sandbox", "runtime_generation": int64(9007199254740993), "lifecycle_epoch": 8,
		"desired_state": "active", "active_lifecycle": false, "observed_at": "2026-10-15T00:00:02Z",
		"runtime": map[string]any{
			"launch_authority": check("available", "external_launch_authority_present", "regional_authority"),
			"liveness":         check("available", "runsc_running", "node_runtime"),
		},
		"execution": check("unavailable", "command_start_failed", "guest_diagnostic"),
		"terminal":  check("unknown", "guest_diagnostics_unavailable", "none"),
		"storage": map[string]any{
			"identity":         check("available", "rootfs_identity_present", "regional_authority"),
			"writer_authority": check("available", "current_writer_lease_present", "regional_authority"),
			"io":               check("unknown", "storage_io_unobserved", "none"),
		},
	}
}

func TestSandboxHealthIndependentEvidenceAndExactGeneration(t *testing.T) {
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/api/v1/sandboxes/sandbox/health" || r.URL.RawQuery != "" || r.ContentLength > 0 {
			t.Error("unexpected health request")
		}
		writeJSON(t, w, 200, map[string]any{"success": true, "data": healthTestData()})
	})
	defer server.Close()
	health, err := client.GetSandboxHealth(t.Context(), "sandbox")
	if err != nil {
		t.Fatal(err)
	}
	if health.RuntimeGeneration != 9007199254740993 || health.Runtime.Liveness.State != "available" || health.Execution.State != "unavailable" ||
		health.Terminal.State != "unknown" || !health.Terminal.ObservedAt.IsNull() || health.Storage.Io.State != "unknown" || calls != 1 {
		t.Fatal("independent health evidence changed")
	}
	if health.Execution.ObservedAt.Value.Equal(health.ObservedAt) {
		t.Fatal("original observation timestamp replaced")
	}
}

func TestSandboxHealthRejectsMalformedOrForeignObservations(t *testing.T) {
	changes := map[string]func(map[string]any){
		"identity":            func(v map[string]any) { v["sandbox_id"] = "foreign" },
		"generation":          func(v map[string]any) { v["runtime_generation"] = -1 },
		"boolean generation":  func(v map[string]any) { v["runtime_generation"] = true },
		"missing field":       func(v map[string]any) { delete(v, "active_lifecycle") },
		"unknown state":       func(v map[string]any) { v["execution"].(map[string]any)["state"] = "healthy" },
		"raw reason":          func(v map[string]any) { v["execution"].(map[string]any)["reason"] = "private backend error" },
		"unknown source":      func(v map[string]any) { v["execution"].(map[string]any)["source"] = "guest" },
		"null observed":       func(v map[string]any) { v["execution"].(map[string]any)["observed_at"] = nil },
		"missing nullable":    func(v map[string]any) { delete(v["terminal"].(map[string]any), "observed_at") },
		"unsourced available": func(v map[string]any) { v["terminal"].(map[string]any)["state"] = "available" },
		"unsourced timestamp": func(v map[string]any) { v["terminal"].(map[string]any)["observed_at"] = "2026-10-15T00:00:01Z" },
		"invalid date":        func(v map[string]any) { v["observed_at"] = "invalid" },
		"null date":           func(v map[string]any) { v["observed_at"] = nil },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				data := healthTestData()
				change(data)
				writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
			})
			defer server.Close()
			if _, err := client.GetSandboxHealth(t.Context(), "sandbox"); err == nil {
				t.Fatal("invalid health accepted")
			}
		})
	}
}

func TestSandboxHealthNoRetryOrLifecycleFallback(t *testing.T) {
	for _, status := range []int{401, 403, 404, 503} {
		calls := 0
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "unavailable"}})
		})
		if _, err := client.GetSandboxHealth(t.Context(), "sandbox"); err == nil {
			t.Fatal("missing error")
		}
		for _, invalid := range []string{"", " sandbox", "sandbox "} {
			if _, err := client.GetSandboxHealth(t.Context(), invalid); err == nil {
				t.Fatal("invalid identity accepted")
			}
		}
		server.Close()
		if calls != 1 {
			t.Fatal("unexpected request or retry")
		}
	}
}
