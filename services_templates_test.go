package sandbox0

import (
	"context"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestTemplateServiceSuccess(t *testing.T) {
	routes := routeMap{
		routeKey(http.MethodGet, "/api/v1/templates"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"templates": []map[string]any{
						{},
					},
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/templates/tpl-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{},
			})
		},
		routeKey(http.MethodPost, "/api/v1/templates"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{},
			})
		},
		routeKey(http.MethodPut, "/api/v1/templates/tpl-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{},
			})
		},
		routeKey(http.MethodDelete, "/api/v1/templates/tpl-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"message": "deleted",
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/templates/tpl-1/pool/warm"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"message": "warming",
				},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	ctx := context.Background()

	if _, err := client.Templates.List(ctx); err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if _, err := client.Templates.Get(ctx, "tpl-1"); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if _, err := client.Templates.Create(ctx, apispec.SandboxTemplate{}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := client.Templates.Update(ctx, "tpl-1", apispec.SandboxTemplate{}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if _, err := client.Templates.Delete(ctx, "tpl-1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := client.Templates.WarmPool(ctx, "tpl-1", apispec.WarmPoolRequest{}); err != nil {
		t.Fatalf("warm pool failed: %v", err)
	}
}

func TestTemplateServiceNotFound(t *testing.T) {
	routes := routeMap{
		routeKey(http.MethodGet, "/api/v1/templates/tpl-404"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusNotFound, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "not_found",
					"message": "missing",
				},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	if _, err := client.Templates.Get(context.Background(), "tpl-404"); err == nil {
		t.Fatal("expected error for missing template")
	}
}
