package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestDeployFunctionBuildsSnapshotRequest(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/functions/deploy" {
			t.Fatalf("path = %s, want /api/v1/functions/deploy", r.URL.Path)
		}
		req := decodeFunctionDeployRequest(t, r)
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
		if !ok || source.Type != apispec.FunctionSourceTypeSnapshot {
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
		writeFunctionDeployResponse(t, w, http.StatusCreated)
	})
	defer server.Close()

	activate := false
	result, err := client.DeployFunction(context.Background(), FunctionDeploySpec{
		Name:     "API",
		Slug:     "api",
		Template: "node",
		Service: FunctionServiceSpec{
			ID:         "api",
			Port:       8080,
			Command:    []string{"node", "server.js"},
			CWD:        "/app",
			EnvVars:    map[string]string{"NODE_ENV": "production"},
			HealthPath: "/healthz",
		},
		Mounts: []FunctionSnapshotMount{{SnapshotID: "snap_123", MountPath: "/app"}},
		Scale: &apispec.FunctionScalePolicy{
			IdleTimeoutSeconds: apispec.NewOptInt32(120),
		},
		Activate: &activate,
	})
	if err != nil {
		t.Fatalf("DeployFunction() error = %v", err)
	}
	if result.Function.ID != "fn_123" || result.Revision.ID != "fr_123" {
		t.Fatalf("result = %+v, want function/revision IDs", result)
	}
}

func TestDeployFunctionFromSandboxService(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/functions/deploy" {
			t.Fatalf("%s %s, want POST /api/v1/functions/deploy", r.Method, r.URL.Path)
		}
		req := decodeFunctionDeployRequest(t, r)
		source, ok := req.Source.Get()
		if !ok || source.Type != apispec.FunctionSourceTypeSandboxService {
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
		writeFunctionDeployResponse(t, w, http.StatusCreated)
	})
	defer server.Close()

	result, err := client.DeployFunctionFromSandboxService(
		context.Background(),
		"sb_123",
		"api",
		WithFunctionName("API"),
		WithFunctionSlug("api"),
	)
	if err != nil {
		t.Fatalf("DeployFunctionFromSandboxService() error = %v", err)
	}
	if result.Function.Slug != "api" {
		t.Fatalf("function slug = %q, want api", result.Function.Slug)
	}
}

func TestFunctionLifecycleMethods(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/functions":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"functions": []map[string]any{functionResponsePayload()},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/functions/api":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data":    functionResponsePayload(),
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/functions/api/revisions":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"revisions": []map[string]any{revisionResponsePayload()},
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/functions/api/active-revision":
			var req apispec.ActivateFunctionRevisionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode activate request: %v", err)
			}
			if req.RevisionID != "fr_123" {
				t.Fatalf("revision_id = %q, want fr_123", req.RevisionID)
			}
			writeFunctionDeployResponse(t, w, http.StatusOK)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/functions/api":
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

	functions, err := client.ListFunctions(context.Background())
	if err != nil {
		t.Fatalf("ListFunctions() error = %v", err)
	}
	if len(functions) != 1 || functions[0].ID != "fn_123" {
		t.Fatalf("functions = %+v, want fn_123", functions)
	}
	fn, err := client.GetFunction(context.Background(), "api")
	if err != nil {
		t.Fatalf("GetFunction() error = %v", err)
	}
	if fn.Slug != "api" {
		t.Fatalf("function slug = %q, want api", fn.Slug)
	}
	revisions, err := client.ListFunctionRevisions(context.Background(), "api")
	if err != nil {
		t.Fatalf("ListFunctionRevisions() error = %v", err)
	}
	if len(revisions) != 1 || revisions[0].ID != "fr_123" {
		t.Fatalf("revisions = %+v, want fr_123", revisions)
	}
	if _, err := client.ActivateFunctionRevision(context.Background(), "api", "fr_123"); err != nil {
		t.Fatalf("ActivateFunctionRevision() error = %v", err)
	}
	if _, err := client.DeleteFunction(context.Background(), "api"); err != nil {
		t.Fatalf("DeleteFunction() error = %v", err)
	}
}

func decodeFunctionDeployRequest(t *testing.T, r *http.Request) apispec.FunctionDeployRequest {
	t.Helper()
	defer r.Body.Close()

	var req apispec.FunctionDeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatalf("decode function deploy request: %v", err)
	}
	return req
}

func writeFunctionDeployResponse(t *testing.T, w http.ResponseWriter, status int) {
	t.Helper()
	writeJSON(t, w, status, map[string]any{
		"success": true,
		"data": map[string]any{
			"function": functionResponsePayload(),
			"revision": revisionResponsePayload(),
		},
	})
}

func functionResponsePayload() map[string]any {
	return map[string]any{
		"id":                 "fn_123",
		"team_id":            "team_123",
		"name":               "API",
		"slug":               "api",
		"domain_label":       "api-abcd1234",
		"url":                "https://api-abcd1234.fn.us.sandbox0.app",
		"active_revision_id": "fr_123",
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
		"id":              "fr_123",
		"function_id":     "fn_123",
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
