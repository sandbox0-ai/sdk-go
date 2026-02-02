package sandbox0

import (
	"context"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxContextServiceSuccess(t *testing.T) {
	routes := routeMap{
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/contexts"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"contexts": []map[string]any{
						{
							"id":         "ctx-1",
							"created_at": "2024-01-02T00:00:00Z",
							"paused":     false,
							"running":    true,
							"type":       "repl",
						},
					},
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         "ctx-1",
					"created_at": "2024-01-02T00:00:00Z",
					"paused":     false,
					"running":    true,
					"type":       "repl",
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/contexts/ctx-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         "ctx-1",
					"created_at": "2024-01-02T00:00:00Z",
					"paused":     false,
					"running":    true,
					"type":       "repl",
				},
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxes/sb-1/contexts/ctx-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"deleted": true,
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts/ctx-1/restart"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         "ctx-1",
					"created_at": "2024-01-02T00:00:00Z",
					"paused":     false,
					"running":    true,
					"type":       "repl",
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts/ctx-1/input"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"written": true,
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts/ctx-1/exec"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"output": "ok",
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts/ctx-1/resize"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"resized": true,
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts/ctx-1/signal"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"signaled": true,
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/contexts/ctx-1/stats"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	sandbox := client.Sandbox("sb-1")
	ctx := context.Background()

	if _, err := sandbox.Contexts.List(ctx); err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if _, err := sandbox.Contexts.Create(ctx, apispec.CreateContextRequest{}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := sandbox.Contexts.Get(ctx, "ctx-1"); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if _, err := sandbox.Contexts.Delete(ctx, "ctx-1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := sandbox.Contexts.Restart(ctx, "ctx-1"); err != nil {
		t.Fatalf("restart failed: %v", err)
	}
	if _, err := sandbox.Contexts.Input(ctx, "ctx-1", "in"); err != nil {
		t.Fatalf("input failed: %v", err)
	}
	if _, err := sandbox.Contexts.Exec(ctx, "ctx-1", "exec"); err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if _, err := sandbox.Contexts.Resize(ctx, "ctx-1", 10, 20); err != nil {
		t.Fatalf("resize failed: %v", err)
	}
	if _, err := sandbox.Contexts.Signal(ctx, "ctx-1", "SIGTERM"); err != nil {
		t.Fatalf("signal failed: %v", err)
	}
	if _, err := sandbox.Contexts.Stats(ctx, "ctx-1"); err != nil {
		t.Fatalf("stats failed: %v", err)
	}
}
