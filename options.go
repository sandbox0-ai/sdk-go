package sandbox0

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// TokenSource provides bearer tokens for API requests.
type TokenSource func(ctx context.Context) (string, error)

type clientConfig struct {
	baseURL        string
	tokenSource    TokenSource
	httpClient     apispec.HttpRequestDoer
	userAgent      string
	requestEditors []apispec.RequestEditorFn
}

// Option configures a Client.
type Option func(*clientConfig) error

// WithBaseURL overrides the API base URL.
func WithBaseURL(baseURL string) Option {
	return func(cfg *clientConfig) error {
		if baseURL == "" {
			return errors.New("base URL cannot be empty")
		}
		cfg.baseURL = baseURL
		return nil
	}
}

// WithToken sets a static bearer token.
func WithToken(token string) Option {
	return func(cfg *clientConfig) error {
		cfg.tokenSource = func(context.Context) (string, error) {
			return token, nil
		}
		return nil
	}
}

// WithTokenSource sets a dynamic bearer token provider.
func WithTokenSource(source TokenSource) Option {
	return func(cfg *clientConfig) error {
		cfg.tokenSource = source
		return nil
	}
}

// WithHTTPClient sets a custom HTTP client (Doer).
func WithHTTPClient(client apispec.HttpRequestDoer) Option {
	return func(cfg *clientConfig) error {
		cfg.httpClient = client
		return nil
	}
}

// WithTimeout sets the HTTP client timeout.
// If no HTTP client is configured, a new http.Client is created.
func WithTimeout(timeout time.Duration) Option {
	return func(cfg *clientConfig) error {
		if timeout <= 0 {
			return errors.New("timeout must be positive")
		}
		if cfg.httpClient == nil {
			cfg.httpClient = &http.Client{Timeout: timeout}
			return nil
		}
		httpClient, ok := cfg.httpClient.(*http.Client)
		if !ok {
			return errors.New("timeout can only be applied to *http.Client")
		}
		httpClient.Timeout = timeout
		return nil
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(userAgent string) Option {
	return func(cfg *clientConfig) error {
		cfg.userAgent = userAgent
		return nil
	}
}

// WithRequestEditor appends a request editor to all requests.
func WithRequestEditor(editor apispec.RequestEditorFn) Option {
	return func(cfg *clientConfig) error {
		cfg.requestEditors = append(cfg.requestEditors, editor)
		return nil
	}
}
