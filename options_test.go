package sandbox0

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

type testDoer struct{}

func (d testDoer) Do(*http.Request) (*http.Response, error) {
	return nil, context.Canceled
}

func TestWithBaseURLValidation(t *testing.T) {
	cfg := clientConfig{}
	err := WithBaseURL("")(&cfg)
	if err == nil {
		t.Fatal("expected error for empty base URL")
	}
}

func TestWithTimeoutCreatesHTTPClient(t *testing.T) {
	cfg := clientConfig{}
	err := WithTimeout(2 * time.Second)(&cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	httpClient, ok := cfg.httpClient.(*http.Client)
	if !ok {
		t.Fatalf("expected *http.Client, got %T", cfg.httpClient)
	}
	if httpClient.Timeout != 2*time.Second {
		t.Fatalf("expected timeout 2s, got %v", httpClient.Timeout)
	}
}

func TestWithTimeoutRejectsInvalidClient(t *testing.T) {
	cfg := clientConfig{
		httpClient: apispec.HttpRequestDoer(testDoer{}),
	}
	err := WithTimeout(time.Second)(&cfg)
	if err == nil {
		t.Fatal("expected error for non-*http.Client")
	}
}

func TestWithTimeoutRejectsNonPositive(t *testing.T) {
	cfg := clientConfig{}
	err := WithTimeout(0)(&cfg)
	if err == nil {
		t.Fatal("expected error for non-positive timeout")
	}
}
