package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestDeployRunBuildsSnapshotRequest(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/runs/deploy" {
			t.Fatalf("path = %s, want /api/v1/runs/deploy", r.URL.Path)
		}
		req := decodeRunDeployRequest(t, r)
		if name, ok := req.Name.Get(); !ok || name != "API" {
			t.Fatalf("name = %q set=%v, want API", name, ok)
		}
		if slug, ok := req.Slug.Get(); !ok || slug != "api" {
			t.Fatalf("slug = %q set=%v, want api", slug, ok)
		}
		if activate, ok := req.Activate.Get(); !ok || activate {
			t.Fatalf("activate = %v set=%v, want false", activate, ok)
		}
		scale, ok := req.Scale.Get()
		if !ok {
			t.Fatal("scale not set")
		}
		if idle, ok := scale.IdleTimeoutSeconds.Get(); !ok || idle != 120 {
			t.Fatalf("idle timeout = %d set=%v, want 120", idle, ok)
		}
		source, ok := req.Source.Get()
		if !ok || source.Type != apispec.RunSourceTypeSnapshot {
			t.Fatalf("source = %+v set=%v, want snapshot", source, ok)
		}
		spec, ok := req.Spec.Get()
		if !ok {
			t.Fatal("spec not set")
		}
		if spec.Template != "node" {
			t.Fatalf("template = %q, want node", spec.Template)
		}
		if spec.Service.ID != "api" || spec.Service.Port != 8080 {
			t.Fatalf("service = %+v, want api:8080", spec.Service)
		}
		runtime, ok := spec.Service.Runtime.Get()
		if !ok || runtime.Type != apispec.SandboxAppServiceRuntimeTypeCmd {
			t.Fatalf("runtime = %+v set=%v, want cmd", runtime, ok)
		}
		if len(runtime.Command) != 2 || runtime.Command[0] != "node" || runtime.Command[1] != "server.js" {
			t.Fatalf("command = %+v, want node server.js", runtime.Command)
		}
		if cwd, ok := runtime.Cwd.Get(); !ok || cwd != "/app" {
			t.Fatalf("cwd = %q set=%v, want /app", cwd, ok)
		}
		if len(spec.Mounts) != 1 || spec.Mounts[0].SnapshotID != "snap_123" || spec.Mounts[0].MountPath != "/app" {
			t.Fatalf("mounts = %+v, want snap_123:/app", spec.Mounts)
		}
		writeRunDeployResponse(t, w, http.StatusCreated)
	})
	defer server.Close()

	activate := false
	result, err := client.DeployRun(context.Background(), RunDeploySpec{
		Name:     "API",
		Slug:     "api",
		Template: "node",
		Service: RunServiceSpec{
			ID:         "api",
			Port:       8080,
			Command:    []string{"node", "server.js"},
			CWD:        "/app",
			EnvVars:    map[string]string{"NODE_ENV": "production"},
			HealthPath: "/healthz",
		},
		Mounts: []RunSnapshotMount{{SnapshotID: "snap_123", MountPath: "/app"}},
		Scale: &apispec.RunScalePolicy{
			IdleTimeoutSeconds: apispec.NewOptInt32(120),
		},
		Activate: &activate,
	})
	if err != nil {
		t.Fatalf("DeployRun() error = %v", err)
	}
	if result.Run.ID != "run_123" || result.Revision.ID != "rev_123" {
		t.Fatalf("result = %+v, want run/revision IDs", result)
	}
}

func TestDeployRunFromSandboxService(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/runs/deploy" {
			t.Fatalf("%s %s, want POST /api/v1/runs/deploy", r.Method, r.URL.Path)
		}
		req := decodeRunDeployRequest(t, r)
		source, ok := req.Source.Get()
		if !ok || source.Type != apispec.RunSourceTypeSandboxService {
			t.Fatalf("source = %+v set=%v, want sandbox_service", source, ok)
		}
		sandboxSource, ok := source.SandboxService.Get()
		if !ok {
			t.Fatal("sandbox service source not set")
		}
		if sandboxSource.SandboxID != "sb_123" || sandboxSource.ServiceID != "api" {
			t.Fatalf("sandbox source = %+v, want sb_123/api", sandboxSource)
		}
		if req.Spec.IsSet() {
			t.Fatal("sandbox-service deploy should not send spec")
		}
		writeRunDeployResponse(t, w, http.StatusCreated)
	})
	defer server.Close()

	result, err := client.DeployRunFromSandboxService(
		context.Background(),
		"sb_123",
		"api",
		WithRunName("API"),
		WithRunSlug("api"),
	)
	if err != nil {
		t.Fatalf("DeployRunFromSandboxService() error = %v", err)
	}
	if result.Run.Slug != "api" {
		t.Fatalf("run slug = %q, want api", result.Run.Slug)
	}
}

func TestRunLifecycleMethods(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/runs":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"runs": []map[string]any{runResponsePayload()},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/runs/api":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data":    runResponsePayload(),
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/runs/api/revisions":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"revisions": []map[string]any{revisionResponsePayload()},
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/runs/api/active-revision":
			var req apispec.ActivateRunRevisionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode activate request: %v", err)
			}
			if req.RevisionID != "rev_123" {
				t.Fatalf("revision_id = %q, want rev_123", req.RevisionID)
			}
			writeRunDeployResponse(t, w, http.StatusOK)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/runs/api":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"deleted": true,
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	runs, err := client.ListRuns(context.Background())
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs) != 1 || runs[0].ID != "run_123" {
		t.Fatalf("runs = %+v, want run_123", runs)
	}
	run, err := client.GetRun(context.Background(), "api")
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if run.Slug != "api" {
		t.Fatalf("run slug = %q, want api", run.Slug)
	}
	revisions, err := client.ListRunRevisions(context.Background(), "api")
	if err != nil {
		t.Fatalf("ListRunRevisions() error = %v", err)
	}
	if len(revisions) != 1 || revisions[0].ID != "rev_123" {
		t.Fatalf("revisions = %+v, want rev_123", revisions)
	}
	if _, err := client.ActivateRunRevision(context.Background(), "api", "rev_123"); err != nil {
		t.Fatalf("ActivateRunRevision() error = %v", err)
	}
	if _, err := client.DeleteRun(context.Background(), "api"); err != nil {
		t.Fatalf("DeleteRun() error = %v", err)
	}
}

func decodeRunDeployRequest(t *testing.T, r *http.Request) apispec.RunDeployRequest {
	t.Helper()
	defer r.Body.Close()

	var req apispec.RunDeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatalf("decode run deploy request: %v", err)
	}
	return req
}

func writeRunDeployResponse(t *testing.T, w http.ResponseWriter, status int) {
	t.Helper()
	writeJSON(t, w, status, map[string]any{
		"success": true,
		"data": map[string]any{
			"run":      runResponsePayload(),
			"revision": revisionResponsePayload(),
		},
	})
}

func runResponsePayload() map[string]any {
	return map[string]any{
		"id":                 "run_123",
		"team_id":            "team_123",
		"name":               "API",
		"slug":               "api",
		"domain_label":       "api-abcd1234",
		"url":                "https://api-abcd1234.us.sandbox0.run",
		"active_revision_id": "rev_123",
		"enabled":            true,
		"scale": map[string]any{
			"max_instances":           1,
			"target_concurrency":      1,
			"idle_timeout_seconds":    300,
			"startup_timeout_seconds": 90,
		},
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
	}
}

func revisionResponsePayload() map[string]any {
	return map[string]any{
		"id":              "rev_123",
		"run_id":          "run_123",
		"team_id":         "team_123",
		"number":          1,
		"source":          map[string]any{"type": "snapshot"},
		"status":          "active",
		"runtime_sandbox": "sb_runtime",
		"spec": map[string]any{
			"template": "node",
			"service": map[string]any{
				"id":   "api",
				"port": 8080,
				"ingress": map[string]any{
					"public": true,
					"routes": []map[string]any{{"id": "api", "resume": true}},
				},
			},
			"mounts": []map[string]any{},
		},
		"created_at": "2026-01-01T00:00:00Z",
	}
}
