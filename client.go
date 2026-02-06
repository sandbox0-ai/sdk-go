package sandbox0

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

const defaultBaseURL = "https://api.sandbox0.ai"

// Client is the high-level Sandbox0 SDK client.
type Client struct {
	api            *apispec.Client
	baseURL        string
	tokenSource    TokenSource
	userAgent      string
	requestEditors []apispec.RequestEditor
}

// NewClient creates a new Sandbox0 SDK client.
func NewClient(opts ...Option) (*Client, error) {
	cfg := clientConfig{
		baseURL: defaultBaseURL,
	}
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	client := &Client{
		baseURL:        cfg.baseURL,
		tokenSource:    cfg.tokenSource,
		userAgent:      cfg.userAgent,
		requestEditors: cfg.requestEditors,
	}

	var clientOpts []apispec.ClientOption
	if cfg.httpClient != nil {
		clientOpts = append(clientOpts, apispec.WithClient(cfg.httpClient))
	}
	clientOpts = append(clientOpts, apispec.WithRequestEditor(client.applyRequestEditors))
	clientOpts = append(clientOpts, apispec.WithResponseEditor(handleErrorResponse))
	clientOpts = append(clientOpts, apispec.WithResponseEditor(normalizeNullMapResponse))
	for _, editor := range cfg.responseEditors {
		clientOpts = append(clientOpts, apispec.WithResponseEditor(editor))
	}

	securitySource := clientSecuritySource{tokenSource: cfg.tokenSource}
	apiClient, err := apispec.NewClient(cfg.baseURL, securitySource, clientOpts...)
	if err != nil {
		return nil, err
	}

	client.api = apiClient
	return client, nil
}

// API exposes the generated OpenAPI client for low-level access.
func (c *Client) API() *apispec.Client {
	return c.api
}

// Sandbox returns a convenience wrapper for a known sandbox ID.
func (c *Client) Sandbox(id string) *Sandbox {
	return &Sandbox{
		ID:                id,
		client:            c,
		replContextByLang: map[string]string{},
	}
}

func (c *Client) applyRequestEditors(ctx context.Context, req *http.Request) error {
	if c.userAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if c.tokenSource != nil && req.Header.Get("Authorization") == "" {
		token, err := c.tokenSource(ctx)
		if err != nil {
			return err
		}
		if strings.TrimSpace(token) != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}
	}
	for _, editor := range c.requestEditors {
		if err := editor(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

type clientSecuritySource struct {
	tokenSource TokenSource
}

func (s clientSecuritySource) BearerAuth(ctx context.Context, _ apispec.OperationName) (apispec.BearerAuth, error) {
	if s.tokenSource == nil {
		return apispec.BearerAuth{}, ogenerrors.ErrSkipClientSecurity
	}
	token, err := s.tokenSource(ctx)
	if err != nil {
		return apispec.BearerAuth{}, err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return apispec.BearerAuth{}, ogenerrors.ErrSkipClientSecurity
	}
	return apispec.BearerAuth{Token: token}, nil
}

func (c *Client) websocketURL(path string) (string, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(baseURL.Scheme) {
	case "https":
		baseURL.Scheme = "wss"
	case "http":
		baseURL.Scheme = "ws"
	}
	baseURL.Path = strings.TrimSuffix(baseURL.Path, "/") + path
	return baseURL.String(), nil
}
