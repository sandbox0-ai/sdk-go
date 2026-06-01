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
					"public_url":  "https://rs-default-api-abcde--p8080.us.sandbox0.app",
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
	if got, want := resp.Services[0].PublicURL.Value, "https://rs-default-api-abcde--p8080.us.sandbox0.app"; !resp.Services[0].PublicURL.Set || got != want {
		t.Fatalf("public url = %q set=%v, want %q", got, resp.Services[0].PublicURL.Set, want)
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

	_, err = sandbox.UpdateServices(context.Background(), []apispec.SandboxAppService{
		{
			ID: "handler",
			Runtime: apispec.NewOptSandboxAppServiceRuntime(apispec.SandboxAppServiceRuntime{
				Type: apispec.SandboxAppServiceRuntimeTypeFunction,
				Function: apispec.NewOptSandboxFunction(apispec.SandboxFunction{
					Runtime: apispec.SandboxFunctionRuntimePython,
					Handler: apispec.NewOptString("handler"),
					Source: apispec.SandboxFunctionSource{
						Type: apispec.SandboxFunctionSourceTypeInline,
						Code: "def handler(request):\n    return {'status': 200, 'body': 'ok'}\n",
					},
				}),
			}),
			Ingress: apispec.SandboxAppServiceIngress{
				Public: true,
				Routes: []apispec.SandboxAppServiceRoute{
					{ID: "handler", PathPrefix: apispec.NewOptString("/"), Resume: true},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateServices(function) error = %v", err)
	}
	if len(gotPut.Services) != 1 {
		t.Fatalf("function update service count = %d, want 1", len(gotPut.Services))
	}
	runtime, ok := gotPut.Services[0].Runtime.Get()
	if !ok || runtime.Type != apispec.SandboxAppServiceRuntimeTypeFunction {
		t.Fatalf("runtime = %+v set=%v, want function", runtime, ok)
	}
	fn, ok := runtime.Function.Get()
	if !ok {
		t.Fatal("runtime function not set")
	}
	if fn.Runtime != apispec.SandboxFunctionRuntimePython || fn.Source.Type != apispec.SandboxFunctionSourceTypeInline {
		t.Fatalf("function = %+v, want python inline", fn)
	}
	if fn.Source.Code == "" {
		t.Fatal("function source code is empty")
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
