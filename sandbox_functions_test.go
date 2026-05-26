package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestInvokeFunction(t *testing.T) {
	var got apispec.FunctionInvokeRequest
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/sandboxes/sb_123/functions/main/invoke" {
			t.Fatalf("path = %s, want function invoke path", r.URL.Path)
		}
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"status":      201,
				"headers":     map[string][]string{"x-value": {"ok"}},
				"body_base64": "aGk=",
			},
		})
	})
	defer server.Close()

	response, err := client.Sandbox("sb_123").InvokeFunction(context.Background(), "main", apispec.FunctionInvokeRequest{
		Method:     apispec.NewOptString("POST"),
		Path:       apispec.NewOptString("/hello"),
		BodyBase64: apispec.NewOptString("e30="),
	})
	if err != nil {
		t.Fatalf("InvokeFunction() error = %v", err)
	}
	if method, ok := got.Method.Get(); !ok || method != "POST" {
		t.Fatalf("method = %q set=%v, want POST", method, ok)
	}
	if path, ok := got.Path.Get(); !ok || path != "/hello" {
		t.Fatalf("path = %q set=%v, want /hello", path, ok)
	}
	if response.Status != 201 {
		t.Fatalf("status = %d, want 201", response.Status)
	}
	if body, ok := response.BodyBase64.Get(); !ok || body != "aGk=" {
		t.Fatalf("body = %q set=%v, want aGk=", body, ok)
	}
}
