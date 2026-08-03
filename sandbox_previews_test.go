package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxPreviewLifecycle(t *testing.T) {
	var createRequest apispec.SandboxPreviewCreateRequest
	var renewRequest apispec.SandboxPreviewRenewRequest
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/sb_123/previews":
			if err := json.NewDecoder(r.Body).Decode(&createRequest); err != nil {
				t.Fatal(err)
			}
			writePreviewResponse(t, w, http.StatusCreated, "https://bootstrap.example.test")
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/sandboxes/sb_123/previews/preview-1":
			if err := json.NewDecoder(r.Body).Decode(&renewRequest); err != nil {
				t.Fatal(err)
			}
			writePreviewResponse(t, w, http.StatusOK, "https://target.example.test")
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/sandboxes/sb_123/previews/preview-1":
			writeJSON(t, w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"message": "preview grant revoked"}})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	sandbox := client.Sandbox("sb_123")
	grant, err := sandbox.CreatePreview(context.Background(), apispec.SandboxPreviewCreateRequest{
		Port:       3000,
		Protocol:   apispec.NewOptSandboxPreviewCreateRequestProtocol(apispec.SandboxPreviewCreateRequestProtocolHTTP),
		Path:       apispec.NewOptString("/dashboard"),
		TTLSeconds: apispec.NewOptInt32(900),
	})
	if err != nil {
		t.Fatalf("CreatePreview: %v", err)
	}
	if grant.ID != "preview-1" || grant.URL.String() != "https://bootstrap.example.test" {
		t.Fatalf("grant = %#v", grant)
	}
	if createRequest.Port != 3000 || createRequest.Path.Value != "/dashboard" {
		t.Fatalf("create request = %#v", createRequest)
	}

	grant, err = sandbox.RenewPreview(context.Background(), grant.ID, apispec.SandboxPreviewRenewRequest{TTLSeconds: apispec.NewOptInt32(600)})
	if err != nil {
		t.Fatalf("RenewPreview: %v", err)
	}
	if grant.URL.String() != "https://target.example.test" || renewRequest.TTLSeconds.Value != 600 {
		t.Fatalf("renew grant = %#v request = %#v", grant, renewRequest)
	}
	if err := sandbox.RevokePreview(context.Background(), grant.ID); err != nil {
		t.Fatalf("RevokePreview: %v", err)
	}
}

func writePreviewResponse(t *testing.T, w http.ResponseWriter, status int, accessURL string) {
	t.Helper()
	writeJSON(t, w, status, map[string]any{
		"success": true,
		"data": map[string]any{
			"id":                 "preview-1",
			"sandbox_id":         "sb_123",
			"port":               3000,
			"protocol":           "http",
			"url":                accessURL,
			"target_url":         "https://target.example.test",
			"expires_at":         "2026-08-03T00:15:00Z",
			"runtime_generation": 4,
		},
	})
}
