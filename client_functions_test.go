package sandbox0

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestClientFunctions(t *testing.T) {
	t.Run("create from sandbox", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/api/v1/functions" {
				t.Fatalf("path = %s, want /api/v1/functions", r.URL.Path)
			}
			body := decodeFunctionCreateRequest(t, r.Body)
			if body.Source.SandboxID != "sbx-1" || body.Source.ServiceID != "web" {
				t.Fatalf("source = %#v, want sbx-1/web", body.Source)
			}
			if got := body.Name.Or(""); got != "web-fn" {
				t.Fatalf("name = %q, want web-fn", got)
			}
			autoscaling, ok := body.Autoscaling.Get()
			if !ok {
				t.Fatal("autoscaling missing")
			}
			if autoscaling.MinWarm != 1 || autoscaling.MaxActive != 3 || autoscaling.TargetConcurrency != 7 || autoscaling.ScaleDownAfterSeconds != 60 {
				t.Fatalf("autoscaling = %#v, want 1/3/7/60", autoscaling)
			}
			writeJSON(t, w, http.StatusCreated, functionCreateResponse())
		})
		defer server.Close()

		result, err := client.CreateFunctionFromSandbox(
			context.Background(),
			"sbx-1",
			"web",
			WithFunctionName("web-fn"),
			WithFunctionAutoscaling(FunctionAutoscaling(1, 3, 7, 60)),
		)
		if err != nil {
			t.Fatalf("CreateFunctionFromSandbox() error = %v", err)
		}
		if result.Function.ID != "fn-1" {
			t.Fatalf("function id = %q, want fn-1", result.Function.ID)
		}
		if result.Revision.RevisionNumber != 1 {
			t.Fatalf("revision number = %d, want 1", result.Revision.RevisionNumber)
		}
		if result.Alias.Alias != "production" {
			t.Fatalf("alias = %q, want production", result.Alias.Alias)
		}
	})

	t.Run("list and get", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/functions":
				writeJSON(t, w, http.StatusOK, map[string]any{
					"success": true,
					"data": map[string]any{
						"functions": []any{functionRecord()},
					},
				})
			case "/api/v1/functions/fn-1":
				writeJSON(t, w, http.StatusOK, map[string]any{
					"success": true,
					"data": map[string]any{
						"function": functionRecord(),
					},
				})
			default:
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
		})
		defer server.Close()

		functions, err := client.ListFunctions(context.Background())
		if err != nil {
			t.Fatalf("ListFunctions() error = %v", err)
		}
		if len(functions) != 1 || functions[0].Host != "web.sandbox0.site" {
			t.Fatalf("functions = %#v, want web.sandbox0.site", functions)
		}

		fn, err := client.GetFunction(context.Background(), "fn-1")
		if err != nil {
			t.Fatalf("GetFunction() error = %v", err)
		}
		if fn.Slug != "web" {
			t.Fatalf("slug = %q, want web", fn.Slug)
		}
	})

	t.Run("update and delete", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && r.URL.Path == "/api/v1/functions/fn-1":
				var body apispec.FunctionUpdateRequest
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode update request: %v", err)
				}
				if got := body.Name.Or(""); got != "new-name" {
					t.Fatalf("name = %q, want new-name", got)
				}
				if enabled := body.Enabled.Or(true); enabled {
					t.Fatalf("enabled = true, want false")
				}
				autoscaling, ok := body.Autoscaling.Get()
				if !ok {
					t.Fatal("autoscaling missing")
				}
				if autoscaling.MinWarm != 0 || autoscaling.MaxActive != 5 || autoscaling.TargetConcurrency != 10 || autoscaling.ScaleDownAfterSeconds != 120 {
					t.Fatalf("autoscaling = %#v, want 0/5/10/120", autoscaling)
				}
				record := functionRecord()
				record["name"] = "new-name"
				record["enabled"] = false
				writeJSON(t, w, http.StatusOK, map[string]any{
					"success": true,
					"data": map[string]any{
						"function": record,
					},
				})
			case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/functions/fn-1":
				record := functionRecord()
				record["deleted_at"] = "2026-05-14T01:00:00Z"
				writeJSON(t, w, http.StatusOK, map[string]any{
					"success": true,
					"data": map[string]any{
						"function": record,
					},
				})
			default:
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
			}
		})
		defer server.Close()

		updated, err := client.UpdateFunctionWithOptions(
			context.Background(),
			"fn-1",
			WithFunctionUpdateName("new-name"),
			WithFunctionEnabled(false),
			WithFunctionUpdateAutoscaling(FunctionAutoscaling(0, 5, 10, 120)),
		)
		if err != nil {
			t.Fatalf("UpdateFunctionWithOptions() error = %v", err)
		}
		if updated.Enabled {
			t.Fatal("updated.Enabled = true, want false")
		}

		deleted, err := client.DeleteFunction(context.Background(), "fn-1")
		if err != nil {
			t.Fatalf("DeleteFunction() error = %v", err)
		}
		if _, ok := deleted.DeletedAt.Get(); !ok {
			t.Fatal("DeletedAt missing")
		}
	})

	t.Run("revision and alias", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/functions/fn-1/revisions":
				writeJSON(t, w, http.StatusOK, map[string]any{
					"success": true,
					"data": map[string]any{
						"revisions": []any{functionRevision(1)},
					},
				})
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/functions/fn-1/revisions/2":
				writeJSON(t, w, http.StatusOK, map[string]any{
					"success": true,
					"data": map[string]any{
						"revision": functionRevision(2),
					},
				})
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/functions/fn-1/revisions":
				body := decodeFunctionRevisionCreateRequest(t, r.Body)
				if body.Source.ServiceID != "web-v2" {
					t.Fatalf("service id = %q, want web-v2", body.Source.ServiceID)
				}
				if promote := body.Promote.Or(true); promote {
					t.Fatalf("promote = true, want false")
				}
				writeJSON(t, w, http.StatusCreated, map[string]any{
					"success": true,
					"data": map[string]any{
						"revision": functionRevision(2),
						"promoted": false,
					},
				})
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/functions/fn-1/aliases":
				writeJSON(t, w, http.StatusOK, map[string]any{
					"success": true,
					"data": map[string]any{
						"aliases": []any{functionAlias(1)},
					},
				})
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/functions/fn-1/aliases/production":
				writeJSON(t, w, http.StatusOK, map[string]any{
					"success": true,
					"data": map[string]any{
						"alias": functionAlias(1),
					},
				})
			case r.Method == http.MethodPut && r.URL.Path == "/api/v1/functions/fn-1/aliases/production":
				var body apispec.FunctionAliasUpdateRequest
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode alias request: %v", err)
				}
				if body.RevisionNumber != 2 {
					t.Fatalf("revision number = %d, want 2", body.RevisionNumber)
				}
				writeJSON(t, w, http.StatusOK, map[string]any{
					"success": true,
					"data": map[string]any{
						"alias": functionAlias(2),
					},
				})
			default:
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
			}
		})
		defer server.Close()

		revisions, err := client.ListFunctionRevisions(context.Background(), "fn-1")
		if err != nil {
			t.Fatalf("ListFunctionRevisions() error = %v", err)
		}
		if len(revisions) != 1 {
			t.Fatalf("len(revisions) = %d, want 1", len(revisions))
		}

		revision, err := client.GetFunctionRevision(context.Background(), "fn-1", 2)
		if err != nil {
			t.Fatalf("GetFunctionRevision() error = %v", err)
		}
		if revision.RevisionNumber != 2 {
			t.Fatalf("revision number = %d, want 2", revision.RevisionNumber)
		}

		created, err := client.CreateFunctionRevisionFromSandbox(context.Background(), "fn-1", "sbx-1", "web-v2", WithFunctionRevisionPromote(false))
		if err != nil {
			t.Fatalf("CreateFunctionRevisionFromSandbox() error = %v", err)
		}
		if created.Promoted {
			t.Fatal("Promoted = true, want false")
		}

		aliases, err := client.ListFunctionAliases(context.Background(), "fn-1")
		if err != nil {
			t.Fatalf("ListFunctionAliases() error = %v", err)
		}
		if len(aliases) != 1 {
			t.Fatalf("len(aliases) = %d, want 1", len(aliases))
		}

		gotAlias, err := client.GetFunctionAlias(context.Background(), "fn-1", "production")
		if err != nil {
			t.Fatalf("GetFunctionAlias() error = %v", err)
		}
		if gotAlias.Alias != "production" {
			t.Fatalf("alias = %q, want production", gotAlias.Alias)
		}

		alias, err := client.SetFunctionAlias(context.Background(), "fn-1", "production", 2)
		if err != nil {
			t.Fatalf("SetFunctionAlias() error = %v", err)
		}
		if alias.RevisionNumber != 2 {
			t.Fatalf("alias revision = %d, want 2", alias.RevisionNumber)
		}
	})

	t.Run("runtime lifecycle", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/functions/fn-1/runtime":
				writeJSON(t, w, http.StatusOK, functionRuntimeResponse("active"))
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/functions/fn-1/runtime/restart":
				writeJSON(t, w, http.StatusOK, functionRuntimeResponse("idle"))
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/functions/fn-1/runtime/recycle":
				writeJSON(t, w, http.StatusOK, functionRuntimeResponse("idle"))
			default:
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
			}
		})
		defer server.Close()

		runtime, err := client.GetFunctionRuntime(context.Background(), "fn-1")
		if err != nil {
			t.Fatalf("GetFunctionRuntime() error = %v", err)
		}
		if runtime.State != apispec.FunctionRuntimeStateActive {
			t.Fatalf("runtime state = %q, want active", runtime.State)
		}

		restarted, err := client.RestartFunctionRuntime(context.Background(), "fn-1")
		if err != nil {
			t.Fatalf("RestartFunctionRuntime() error = %v", err)
		}
		if restarted.State != apispec.FunctionRuntimeStateIdle {
			t.Fatalf("restart state = %q, want idle", restarted.State)
		}

		recycled, err := client.RecycleFunctionRuntime(context.Background(), "fn-1")
		if err != nil {
			t.Fatalf("RecycleFunctionRuntime() error = %v", err)
		}
		if recycled.State != apispec.FunctionRuntimeStateIdle {
			t.Fatalf("recycle state = %q, want idle", recycled.State)
		}
	})
}

func decodeFunctionCreateRequest(t *testing.T, body io.Reader) apispec.FunctionCreateRequest {
	t.Helper()

	var req apispec.FunctionCreateRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return req
}

func decodeFunctionRevisionCreateRequest(t *testing.T, body io.Reader) apispec.FunctionRevisionCreateRequest {
	t.Helper()

	var req apispec.FunctionRevisionCreateRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return req
}

func functionCreateResponse() map[string]any {
	return map[string]any{
		"success": true,
		"data": map[string]any{
			"function": functionRecord(),
			"revision": functionRevision(1),
			"alias":    functionAlias(1),
		},
	}
}

func functionRecord() map[string]any {
	return map[string]any{
		"id":                 "fn-1",
		"team_id":            "team-1",
		"name":               "web",
		"slug":               "web",
		"domain_label":       "web",
		"active_revision_id": "rev-1",
		"enabled":            true,
		"autoscaling":        functionAutoscalingMap(),
		"created_at":         "2026-05-14T00:00:00Z",
		"updated_at":         "2026-05-14T00:00:00Z",
		"host":               "web.sandbox0.site",
		"url":                "https://web.sandbox0.site",
	}
}

func functionRevision(number int) map[string]any {
	return map[string]any{
		"id":                 "rev-1",
		"function_id":        "fn-1",
		"team_id":            "team-1",
		"revision_number":    number,
		"source_sandbox_id":  "sbx-1",
		"source_service_id":  "web",
		"source_template_id": "default",
		"restore_mounts":     []any{},
		"service_snapshot": map[string]any{
			"id":   "web",
			"port": 8080,
			"ingress": map[string]any{
				"public": true,
			},
		},
		"created_at": "2026-05-14T00:00:00Z",
	}
}

func functionAlias(number int) map[string]any {
	return map[string]any{
		"function_id":     "fn-1",
		"alias":           "production",
		"revision_id":     "rev-1",
		"revision_number": number,
		"updated_at":      "2026-05-14T00:00:00Z",
	}
}

func functionRuntimeResponse(state string) map[string]any {
	runtime := map[string]any{
		"function_id":        "fn-1",
		"revision_id":        "rev-1",
		"revision_number":    1,
		"state":              state,
		"autoscaling":        functionAutoscalingMap(),
		"runtime_updated_at": "2026-05-14T00:00:00Z",
		"instances":          []any{},
	}
	if state == "active" {
		runtime["runtime_sandbox_id"] = "sb-runtime"
		runtime["runtime_context_id"] = "ctx-runtime"
	}
	return map[string]any{
		"success": true,
		"data": map[string]any{
			"runtime": runtime,
		},
	}
}

func functionAutoscalingMap() map[string]any {
	return map[string]any{
		"min_warm":                 0,
		"max_active":               20,
		"target_concurrency":       80,
		"scale_down_after_seconds": 300,
	}
}
