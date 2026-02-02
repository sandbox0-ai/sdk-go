package sandbox0

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestApplyRequestEditors(t *testing.T) {
	client := &Client{
		userAgent: "sandbox0-test",
		tokenSource: func(context.Context) (string, error) {
			return "token-123", nil
		},
		requestEditors: []apispec.RequestEditorFn{
			func(_ context.Context, req *http.Request) error {
				req.Header.Set("X-Custom", "ok")
				return nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	if err := client.applyRequestEditors(context.Background(), req); err != nil {
		t.Fatalf("applyRequestEditors failed: %v", err)
	}
	if got := req.Header.Get("User-Agent"); got != "sandbox0-test" {
		t.Fatalf("expected user-agent, got %q", got)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer token-123" {
		t.Fatalf("expected authorization, got %q", got)
	}
	if got := req.Header.Get("X-Custom"); got != "ok" {
		t.Fatalf("expected custom header, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("User-Agent", "existing")
	req.Header.Set("Authorization", "Bearer existing")
	if err := client.applyRequestEditors(context.Background(), req); err != nil {
		t.Fatalf("applyRequestEditors failed: %v", err)
	}
	if got := req.Header.Get("User-Agent"); got != "existing" {
		t.Fatalf("expected existing user-agent, got %q", got)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer existing" {
		t.Fatalf("expected existing authorization, got %q", got)
	}
}

func TestApplyRequestEditorsTokenError(t *testing.T) {
	client := &Client{
		tokenSource: func(context.Context) (string, error) {
			return "", context.Canceled
		},
	}
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	if err := client.applyRequestEditors(context.Background(), req); err == nil {
		t.Fatal("expected token source error")
	}
}

func TestWebsocketURL(t *testing.T) {
	client := &Client{baseURL: "https://example.com/api/"}
	got, err := client.websocketURL("/ws")
	if err != nil {
		t.Fatalf("websocketURL failed: %v", err)
	}
	if got != "wss://example.com/api/ws" {
		t.Fatalf("unexpected websocket URL: %s", got)
	}

	client = &Client{baseURL: "http://example.com"}
	got, err = client.websocketURL("/stream")
	if err != nil {
		t.Fatalf("websocketURL failed: %v", err)
	}
	if got != "ws://example.com/stream" {
		t.Fatalf("unexpected websocket URL: %s", got)
	}
}
