package sandbox0

import (
	"context"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxServiceSuccess(t *testing.T) {
	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxes"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"sandbox_id": "sb-1",
					"template":   "tpl-1",
					"status":     "running",
					"pod_name":   "pod-1",
					"cluster_id": "cluster-1",
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":          "sb-1",
					"pod_name":    "pod-1",
					"status":      "running",
					"team_id":     "team-1",
					"template_id": "tpl-1",
					"claimed_at":  "2024-01-02T00:00:00Z",
					"created_at":  "2024-01-02T00:00:00Z",
					"expires_at":  "2024-01-02T01:00:00Z",
				},
			})
		},
		routeKey(http.MethodPatch, "/api/v1/sandboxes/sb-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":          "sb-1",
					"pod_name":    "pod-1",
					"status":      "running",
					"team_id":     "team-1",
					"template_id": "tpl-1",
					"claimed_at":  "2024-01-02T00:00:00Z",
					"created_at":  "2024-01-02T00:00:00Z",
					"expires_at":  "2024-01-02T01:00:00Z",
				},
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxes/sb-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"message": "deleted",
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/status"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"sandbox_id": "sb-1",
					"status":     "running",
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/pause"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"sandbox_id": "sb-1",
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/resume"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"sandbox_id": "sb-1",
					"resumed":    true,
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/refresh"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"sandbox_id": "sb-1",
					"expires_at": "2024-01-02T01:00:00Z",
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/network"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{},
			})
		},
		routeKey(http.MethodPatch, "/api/v1/sandboxes/sb-1/network"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/bandwidth"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{},
			})
		},
		routeKey(http.MethodPatch, "/api/v1/sandboxes/sb-1/bandwidth"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	ctx := context.Background()

	if _, err := client.Sandboxes.Claim(ctx, "tpl-1"); err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if _, err := client.Sandboxes.Get(ctx, "sb-1"); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if _, err := client.Sandboxes.Update(ctx, "sb-1", apispec.SandboxUpdateRequest{}); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if _, err := client.Sandboxes.Delete(ctx, "sb-1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := client.Sandboxes.Status(ctx, "sb-1"); err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if _, err := client.Sandboxes.Pause(ctx, "sb-1"); err != nil {
		t.Fatalf("pause failed: %v", err)
	}
	if _, err := client.Sandboxes.Resume(ctx, "sb-1"); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if _, err := client.Sandboxes.Refresh(ctx, "sb-1", nil); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if _, err := client.Sandboxes.Refresh(ctx, "sb-1", &apispec.RefreshRequest{RefreshToken: "tok"}); err != nil {
		t.Fatalf("refresh with body failed: %v", err)
	}
	if _, err := client.Sandboxes.GetNetworkPolicy(ctx, "sb-1"); err != nil {
		t.Fatalf("get network policy failed: %v", err)
	}
	if _, err := client.Sandboxes.UpdateNetworkPolicy(ctx, "sb-1", apispec.TplSandboxNetworkPolicy{}); err != nil {
		t.Fatalf("update network policy failed: %v", err)
	}
	if _, err := client.Sandboxes.GetBandwidthPolicy(ctx, "sb-1"); err != nil {
		t.Fatalf("get bandwidth policy failed: %v", err)
	}
	if _, err := client.Sandboxes.UpdateBandwidthPolicy(ctx, "sb-1", apispec.BandwidthPolicySpec{}); err != nil {
		t.Fatalf("update bandwidth policy failed: %v", err)
	}
}

func TestSandboxServiceErrors(t *testing.T) {
	routes := routeMap{
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-403"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusForbidden, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "forbidden",
					"message": "no access",
				},
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxes/sb-404"): func(w http.ResponseWriter, r *http.Request) {
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
	ctx := context.Background()

	if _, err := client.Sandboxes.Get(ctx, "sb-403"); err == nil {
		t.Fatal("expected forbidden error")
	}
	if _, err := client.Sandboxes.Delete(ctx, "sb-404"); err == nil {
		t.Fatal("expected not found error")
	}
}
