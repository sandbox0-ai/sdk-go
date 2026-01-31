package sandbox0

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

const defaultBaseURL = "https://api.sandbox0.ai"

// Client is the high-level Sandbox0 SDK client.
type Client struct {
	api            *apispec.ClientWithResponses
	baseURL        string
	tokenSource    TokenSource
	userAgent      string
	requestEditors []apispec.RequestEditorFn

	Sandboxes *SandboxService
	Volumes   *VolumeService
	Templates *TemplateService
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
		clientOpts = append(clientOpts, apispec.WithHTTPClient(cfg.httpClient))
	}
	clientOpts = append(clientOpts, apispec.WithRequestEditorFn(client.applyRequestEditors))

	apiClient, err := apispec.NewClientWithResponses(cfg.baseURL, clientOpts...)
	if err != nil {
		return nil, err
	}

	client.api = apiClient
	client.Sandboxes = &SandboxService{client: client}
	client.Volumes = &VolumeService{client: client}
	client.Templates = &TemplateService{client: client}

	return client, nil
}

// API exposes the generated OpenAPI client for low-level access.
func (c *Client) API() *apispec.ClientWithResponses {
	return c.api
}

// Sandbox returns a convenience wrapper for a known sandbox ID.
func (c *Client) Sandbox(id string) *Sandbox {
	sandbox := &Sandbox{
		ID:                id,
		client:            c,
		replContextByLang: map[string]string{},
	}
	sandbox.Contexts = SandboxContextService{sandbox: sandbox}
	sandbox.Files = SandboxFileService{sandbox: sandbox}
	return sandbox
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
