//go:build e2e

package sandbox0_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
)

func TestNewClientOptionsAndAPI(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)

	requestCalled := false
	responseCalled := false
	seenUserAgent := ""

	reqEditor := func(_ context.Context, req *http.Request) error {
		requestCalled = true
		seenUserAgent = req.Header.Get("User-Agent")
		return nil
	}
	respEditor := func(_ context.Context, _ *http.Response) error {
		responseCalled = true
		return nil
	}

	httpClient := &http.Client{}
	client, err := sandbox0.NewClient(
		sandbox0.WithBaseURL(cfg.baseURL),
		sandbox0.WithTokenSource(func(context.Context) (string, error) {
			return token, nil
		}),
		sandbox0.WithHTTPClient(httpClient),
		sandbox0.WithTimeout(15*time.Second),
		sandbox0.WithUserAgent("sdk-go-e2e"),
		sandbox0.WithRequestEditor(reqEditor),
		sandbox0.WithResponseEditor(respEditor),
	)
	if err != nil {
		t.Fatalf("create client failed: %v", err)
	}

	if client.API() == nil {
		t.Fatalf("API client should not be nil")
	}
	if client.Sandbox("sandbox-id") == nil {
		t.Fatalf("Sandbox wrapper should not be nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := client.ListTemplate(ctx); err != nil {
		t.Fatalf("list templates failed: %v", err)
	}
	if !requestCalled {
		t.Fatalf("request editor was not invoked")
	}
	if !responseCalled {
		t.Fatalf("response editor was not invoked")
	}
	if seenUserAgent != "sdk-go-e2e" {
		t.Fatalf("expected User-Agent to be set, got %q", seenUserAgent)
	}
}

func TestWithToken(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)

	client := newClientWithToken(t, cfg, token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := client.ListTemplate(ctx); err != nil {
		t.Fatalf("list templates failed: %v", err)
	}
}
