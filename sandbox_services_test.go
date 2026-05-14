package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxServicesLifecycle(t *testing.T) {
	var gotPut apispec.SandboxServicesUpdateRequest
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sandboxes/sb_123/services" {
			t.Fatalf("path = %s, want /api/v1/sandboxes/sb_123/services", r.URL.Path)
		}
		switch r.Method {
		case http.MethodGet:
			writeServicesResponse(t, w, []map[string]any{
				{
					"id":   "api",
					"port": 8080,
					"ingress": map[string]any{
						"public": true,
						"routes": []map[string]any{{"id": "api", "resume": true}},
					},
					"publishable": false,
				},
			})
		case http.MethodPut:
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&gotPut); err != nil {
				t.Fatalf("decode put body: %v", err)
			}
			writeServicesResponse(t, w, nil)
		default:
			t.Fatalf("method = %s, want GET or PUT", r.Method)
		}
	})
	defer server.Close()

	sandbox := client.Sandbox("sb_123")
	resp, err := sandbox.GetServices(context.Background())
	if err != nil {
		t.Fatalf("GetServices() error = %v", err)
	}
	if resp.SandboxID != "sb_123" || len(resp.Services) != 1 {
		t.Fatalf("response = %+v, want one service", resp)
	}

	resp, err = sandbox.ClearServices(context.Background())
	if err != nil {
		t.Fatalf("ClearServices() error = %v", err)
	}
	if gotPut.Services == nil {
		t.Fatal("clear request services = nil, want empty slice")
	}
	if len(resp.Services) != 0 {
		t.Fatalf("clear response services count = %d, want 0", len(resp.Services))
	}
}

func writeServicesResponse(t *testing.T, w http.ResponseWriter, services []map[string]any) {
	t.Helper()
	if services == nil {
		services = []map[string]any{}
	}
	writeJSON(t, w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"sandbox_id": "sb_123",
			"services":   services,
		},
	})
}
