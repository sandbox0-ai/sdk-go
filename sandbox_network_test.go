package sandbox0

import (
	"context"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxNetworkPolicy(t *testing.T) {
	sandboxID := "sb-1"
	policy := newNetworkPolicy()
	routes := routeMap{
		routeKey(http.MethodGet, "/api/v1/sandboxes/"+sandboxID+"/network"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessSandboxNetworkPolicyResponse{
				Success: apispec.SuccessSandboxNetworkPolicyResponseSuccessTrue,
				Data:    apispec.NewOptTplSandboxNetworkPolicy(policy),
			})
		},
		routeKey(http.MethodPatch, "/api/v1/sandboxes/"+sandboxID+"/network"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessSandboxNetworkPolicyResponse{
				Success: apispec.SuccessSandboxNetworkPolicyResponseSuccessTrue,
				Data:    apispec.NewOptTplSandboxNetworkPolicy(policy),
			})
		},
	}

	server := newTestServer(t, routes)
	defer server.Close()
	client := newTestClient(t, server)
	sandbox := client.Sandbox(sandboxID)

	if _, err := sandbox.GetNetworkPolicy(context.Background()); err != nil {
		t.Fatalf("get network policy failed: %v", err)
	}
	if _, err := sandbox.UpdateNetworkPolicy(context.Background(), policy); err != nil {
		t.Fatalf("update network policy failed: %v", err)
	}
}
