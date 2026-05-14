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
			writeJSON(t, w, http.StatusCreated, functionCreateResponse())
		})
		defer server.Close()

		result, err := client.CreateFunctionFromSandbox(context.Background(), "sbx-1", "web", WithFunctionName("web-fn"))
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

		created, err := client.CreateFunctionRevisionFromSandbox(context.Background(), "fn-1", "sbx-1", "web-v2", WithFunctionRevisionPromote(false))
		if err != nil {
			t.Fatalf("CreateFunctionRevisionFromSandbox() error = %v", err)
		}
		if created.Promoted {
			t.Fatal("Promoted = true, want false")
		}

		alias, err := client.SetFunctionAlias(context.Background(), "fn-1", "production", 2)
		if err != nil {
			t.Fatalf("SetFunctionAlias() error = %v", err)
		}
		if alias.RevisionNumber != 2 {
			t.Fatalf("alias revision = %d, want 2", alias.RevisionNumber)
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
