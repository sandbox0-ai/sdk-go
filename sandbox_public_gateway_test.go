package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxPublicGatewayLifecycle(t *testing.T) {
	var gotPut apispec.PublicGatewayConfig
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sandboxes/sb_123/public-gateway" {
			t.Fatalf("path = %s, want /api/v1/sandboxes/sb_123/public-gateway", r.URL.Path)
		}
		switch r.Method {
		case http.MethodGet:
			writePublicGatewayResponse(t, w, true)
		case http.MethodPut:
			defer r.Body.Close()
			if err := json.NewDecoder(r.Body).Decode(&gotPut); err != nil {
				t.Fatalf("decode put body: %v", err)
			}
			writePublicGatewayResponse(t, w, gotPut.Enabled)
		default:
			t.Fatalf("method = %s, want GET or PUT", r.Method)
		}
	})
	defer server.Close()

	sandbox := client.Sandbox("sb_123")
	resp, err := sandbox.GetPublicGateway(context.Background())
	if err != nil {
		t.Fatalf("GetPublicGateway() error = %v", err)
	}
	if !resp.PublicGateway.Enabled || resp.ExposureDomain != "example.test" {
		t.Fatalf("response = %+v, want enabled policy with exposure domain", resp)
	}

	resp, err = sandbox.ClearPublicGateway(context.Background())
	if err != nil {
		t.Fatalf("ClearPublicGateway() error = %v", err)
	}
	if gotPut.Enabled {
		t.Fatal("clear request enabled = true, want false")
	}
	if gotPut.Routes == nil {
		t.Fatal("clear request routes = nil, want empty slice")
	}
	if resp.PublicGateway.Enabled {
		t.Fatal("clear response enabled = true, want false")
	}
}

func writePublicGatewayResponse(t *testing.T, w http.ResponseWriter, enabled bool) {
	t.Helper()
	writeJSON(t, w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"sandbox_id": "sb_123",
			"public_gateway": map[string]any{
				"enabled": enabled,
				"routes": []map[string]any{
					{
						"id":     "api",
						"port":   8080,
						"resume": true,
					},
				},
			},
			"exposure_domain": "example.test",
		},
	})
}
