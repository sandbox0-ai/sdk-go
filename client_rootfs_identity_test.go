package sandbox0

import (
	"net/http"
	"testing"
)

func rootFSIdentityTestData() map[string]any {
	return map[string]any{"sandbox_id": "sandbox", "runtime_generation": int64(7), "lifecycle_epoch": int64(8), "desired_state": "paused", "managed_files_size_bytes": int64(0), "filesystem_id": "filesystem", "head_generation_id": "head", "writer_epoch": int64(9007199254740993), "base_artifact_digest": "sha256:base", "source_oci_digest": "sha256:image", "has_imported_filesystem": true, "active_lifecycle": false, "sandbox_created_at": "2026-10-15T00:00:00Z", "filesystem_created_at": "2026-10-15T00:00:00Z", "observed_at": "2026-10-15T00:00:01Z"}
}
func TestRootFSIdentityUsesOnlyExternalReadAndExactIntegers(t *testing.T) {
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Path != "/api/v1/sandboxes/sandbox/rootfs/identity" || r.URL.RawQuery != "" {
			t.Error("unexpected request")
		}
		writeJSON(t, w, 200, map[string]any{"success": true, "data": rootFSIdentityTestData()})
	})
	defer server.Close()
	identity, err := client.GetSandboxRootFSIdentity(t.Context(), "sandbox")
	if err != nil {
		t.Fatal(err)
	}
	if identity.WriterEpoch != 9007199254740993 || !identity.HasImportedFilesystem || identity.FilesystemID != "filesystem" || calls != 1 {
		t.Fatal("identity changed")
	}
}
func TestRootFSIdentityNeverFallsBackOnMissingOrUnavailable(t *testing.T) {
	for _, status := range []int{404, 409, 503} {
		calls := 0
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "unavailable"}})
		})
		if _, err := client.GetSandboxRootFSIdentity(t.Context(), "sandbox"); err == nil {
			t.Fatal("missing error")
		}
		server.Close()
		if calls != 1 {
			t.Fatal("unexpected retry")
		}
	}
}
func TestRootFSIdentityRejectsMismatchedOrIncompleteResult(t *testing.T) {
	for _, change := range []func(map[string]any){func(v map[string]any) { v["sandbox_id"] = "foreign" }, func(v map[string]any) { v["filesystem_id"] = "" }, func(v map[string]any) { v["writer_epoch"] = -1 }, func(v map[string]any) { delete(v, "has_imported_filesystem") }} {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			data := rootFSIdentityTestData()
			change(data)
			writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
		})
		if _, err := client.GetSandboxRootFSIdentity(t.Context(), "sandbox"); err == nil {
			t.Fatal("invalid identity accepted")
		}
		server.Close()
	}
}

func TestRootFSIdentityRuntimeClassIsOptionalAndFenced(t *testing.T) {
	for _, field := range []string{"runtime_compatibility_digest", "retained_runtime_compatibility_digest"} {
		for _, test := range []struct {
			state, compatibility string
			transition, valid    bool
		}{
			{"paused", "", false, true}, {"active", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false, true},
			{"paused", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false, false},
			{"active", "invalid", false, false}, {"active", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true, false},
		} {
			client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				data := rootFSIdentityTestData()
				data["desired_state"] = test.state
				data["active_lifecycle"] = test.transition
				if test.compatibility != "" {
					data[field] = test.compatibility
				}
				writeJSON(t, w, 200, map[string]any{"success": true, "data": data})
			})
			identity, err := client.GetSandboxRootFSIdentity(t.Context(), "sandbox")
			server.Close()
			if (err == nil) != test.valid {
				t.Fatalf("unexpected runtime compatibility admission: %v", err)
			}
			if err == nil {
				compatibility := identity.RuntimeCompatibilityDigest
				if field == "retained_runtime_compatibility_digest" {
					compatibility = identity.RetainedRuntimeCompatibilityDigest
				}
				value, present := compatibility.Get()
				if present != (test.compatibility != "") || value != test.compatibility {
					t.Fatal("runtime compatibility changed")
				}
			}
		}
	}

}
