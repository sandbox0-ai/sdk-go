package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxUpdateNetworkPolicyWithEgressProxy(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/api/v1/sandboxes/sbx_123/network" {
			t.Fatalf("path = %s, want /api/v1/sandboxes/sbx_123/network", r.URL.Path)
		}
		var policy apispec.SandboxNetworkPolicy
		if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
			t.Fatalf("decode network policy request: %v", err)
		}
		egress, ok := policy.Egress.Get()
		if !ok {
			t.Fatal("egress policy not set")
		}
		if len(egress.ProtocolRules) != 1 {
			t.Fatalf("protocolRules count = %d, want 1", len(egress.ProtocolRules))
		}
		protocolRule := egress.ProtocolRules[0]
		if protocolRule.Protocol != apispec.ProtocolRuleProtocolMcp {
			t.Fatalf("protocol rule protocol = %q, want mcp", protocolRule.Protocol)
		}
		mcp, ok := protocolRule.Mcp.Get()
		if !ok {
			t.Fatal("mcp policy not set")
		}
		tools, ok := mcp.Tools.Get()
		if !ok {
			t.Fatal("mcp tools policy not set")
		}
		if len(tools.Allowed) != 1 || tools.Allowed[0] != "read_file" {
			t.Fatalf("allowed tools = %#v, want read_file", tools.Allowed)
		}
		if len(tools.Denied) != 1 || tools.Denied[0] != "run_command" {
			t.Fatalf("denied tools = %#v, want run_command", tools.Denied)
		}
		proxy, ok := egress.Proxy.Get()
		if !ok {
			t.Fatal("egress proxy not set")
		}
		if proxy.Type != apispec.EgressProxyTypeSocks5 {
			t.Fatalf("proxy type = %q, want socks5", proxy.Type)
		}
		if proxy.Address != "proxy.example.com:1080" {
			t.Fatalf("proxy address = %q, want proxy.example.com:1080", proxy.Address)
		}
		credentialRef, ok := proxy.CredentialRef.Get()
		if !ok || credentialRef != "corp-proxy" {
			t.Fatalf("proxy credentialRef = %q, want corp-proxy", credentialRef)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data":    policy,
		})
	})
	defer server.Close()

	sandbox := client.Sandbox("sbx_123")
	ref := "corp-proxy"
	_, err := sandbox.UpdateNetworkPolicy(context.Background(), apispec.SandboxNetworkPolicy{
		Mode: apispec.SandboxNetworkPolicyModeBlockAll,
		Egress: apispec.NewOptNetworkEgressPolicy(apispec.NetworkEgressPolicy{
			TrafficRules: []apispec.TrafficRule{{
				Name:    apispec.NewOptString("allow-private-api"),
				Action:  apispec.TrafficRuleActionAllow,
				Domains: []string{"api.internal.example.com"},
				Ports: []apispec.PortSpec{{
					Port:     443,
					Protocol: apispec.NewOptString("tcp"),
				}},
				AppProtocols: []apispec.TrafficRuleAppProtocol{apispec.TrafficRuleAppProtocolTLS},
			}},
			ProtocolRules: []apispec.ProtocolRule{{
				Name:     apispec.NewOptString("internal-mcp"),
				Protocol: apispec.ProtocolRuleProtocolMcp,
				Domains:  []string{"api.internal.example.com"},
				Ports: []apispec.PortSpec{{
					Port:     443,
					Protocol: apispec.NewOptString("tcp"),
				}},
				TlsMode: apispec.NewOptEgressTLSMode(apispec.EgressTLSModeTerminateReoriginate),
				HttpMatch: apispec.NewOptHTTPMatch(apispec.HTTPMatch{
					Methods: []string{http.MethodPost},
					Paths:   []string{"/mcp"},
				}),
				Mcp: apispec.NewOptMCPProtocolRule(apispec.MCPProtocolRule{
					Tools: apispec.NewOptMCPToolPolicy(apispec.MCPToolPolicy{
						Allowed: []string{"read_file"},
						Denied:  []string{"run_command"},
					}),
				}),
			}},
			Proxy: apispec.NewOptEgressProxyPolicy(apispec.EgressProxyPolicy{
				Type:          apispec.EgressProxyTypeSocks5,
				Address:       "proxy.example.com:1080",
				CredentialRef: apispec.NewOptString(ref),
			}),
		}),
		CredentialBindings: []apispec.CredentialBinding{{
			Ref:       ref,
			SourceRef: "corp-proxy-source",
			Projection: apispec.ProjectionSpec{
				Type:             apispec.CredentialProjectionTypeUsernamePassword,
				UsernamePassword: &apispec.UsernamePasswordProjection{},
			},
		}},
	})
	if err != nil {
		t.Fatalf("UpdateNetworkPolicy() error = %v", err)
	}
}
