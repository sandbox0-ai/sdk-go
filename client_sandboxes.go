package sandbox0

import (
	"context"
	"net/http"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

type sandboxOptions struct {
	config *apispec.SandboxConfig
}

// SandboxOption configures sandbox creation.
type SandboxOption func(*sandboxOptions)

func ensureSandboxConfig(opts *sandboxOptions) *apispec.SandboxConfig {
	if opts.config == nil {
		opts.config = &apispec.SandboxConfig{}
	}
	return opts.config
}

// WithSandboxConfig sets the sandbox configuration for creation.
func WithSandboxConfig(config apispec.SandboxConfig) SandboxOption {
	return func(opts *sandboxOptions) {
		opts.config = &config
	}
}

// WithSandboxTTL sets the soft TTL (seconds) for created sandboxes.
func WithSandboxTTL(ttlSec int32) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		config.Ttl = &ttlSec
	}
}

// WithSandboxHardTTL sets the hard TTL (seconds) for created sandboxes.
func WithSandboxHardTTL(ttlSec int32) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		config.HardTtl = &ttlSec
	}
}

// WithSandboxWebhook configures webhook delivery for sandbox events.
func WithSandboxWebhook(url, secret string) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		if config.Webhook == nil {
			config.Webhook = &apispec.WebhookConfig{}
		}
		config.Webhook.Url = &url
		config.Webhook.Secret = &secret
	}
}

// WithSandboxWebhookWatchDir sets the webhook watch directory (file events).
func WithSandboxWebhookWatchDir(watchDir string) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		if config.Webhook == nil {
			config.Webhook = &apispec.WebhookConfig{}
		}
		config.Webhook.WatchDir = &watchDir
	}
}

// WithSandboxNetworkPolicy sets the sandbox network policy at claim time.
func WithSandboxNetworkPolicy(policy apispec.TplSandboxNetworkPolicy) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		config.Network = &policy
	}
}

// ClaimSandbox creates (claims) a sandbox and returns a convenience wrapper.
func (c *Client) ClaimSandbox(ctx context.Context, template string, opts ...SandboxOption) (*Sandbox, error) {
	options := sandboxOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	req := apispec.ClaimRequest{
		Template: &template,
		Config:   options.config,
	}

	resp, err := c.api.PostApiV1SandboxesWithResponse(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 != nil && resp.JSON201.Data != nil {
		data := resp.JSON201.Data
		sandbox := &Sandbox{
			ID:                data.SandboxId,
			Template:          data.Template,
			ClusterID:         data.ClusterId,
			PodName:           data.PodName,
			Status:            data.Status,
			client:            c,
			replContextByLang: map[string]string{},
		}
		return sandbox, nil
	}
	if resp.JSON400 != nil {
		return nil, apiErrorFromEnvelope(resp.HTTPResponse, resp.JSON400)
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// GetSandbox returns sandbox details by ID.
func (c *Client) GetSandbox(ctx context.Context, sandboxID string) (*apispec.Sandbox, error) {
	resp, err := c.api.GetApiV1SandboxesIdWithResponse(ctx, apispec.SandboxID(sandboxID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	if resp.JSON403 != nil {
		return nil, apiErrorFromEnvelope(resp.HTTPResponse, resp.JSON403)
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// UpdateSandbox updates sandbox configuration.
func (c *Client) UpdateSandbox(ctx context.Context, sandboxID string, request apispec.SandboxUpdateRequest) (*apispec.Sandbox, error) {
	resp, err := c.api.PatchApiV1SandboxesIdWithResponse(ctx, apispec.SandboxID(sandboxID), request)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	if resp.JSON400 != nil {
		return nil, apiErrorFromEnvelope(resp.HTTPResponse, resp.JSON400)
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// DeleteSandbox terminates a sandbox.
func (c *Client) DeleteSandbox(ctx context.Context, sandboxID string) (*apispec.SuccessMessageResponse, error) {
	resp, err := c.api.DeleteApiV1SandboxesIdWithResponse(ctx, apispec.SandboxID(sandboxID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	if resp.JSON403 != nil {
		return nil, apiErrorFromEnvelope(resp.HTTPResponse, resp.JSON403)
	}
	if resp.JSON404 != nil {
		return nil, apiErrorFromEnvelope(resp.HTTPResponse, resp.JSON404)
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// StatusSandbox returns the sandbox status.
func (c *Client) StatusSandbox(ctx context.Context, sandboxID string) (*apispec.SandboxStatus, error) {
	resp, err := c.api.GetApiV1SandboxesIdStatusWithResponse(ctx, apispec.SandboxID(sandboxID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// PauseSandbox suspends a sandbox.
func (c *Client) PauseSandbox(ctx context.Context, sandboxID string) (*apispec.PauseSandboxResponse, error) {
	resp, err := c.api.PostApiV1SandboxesIdPauseWithResponse(ctx, apispec.SandboxID(sandboxID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// ResumeSandbox resumes a sandbox.
func (c *Client) ResumeSandbox(ctx context.Context, sandboxID string) (*apispec.ResumeSandboxResponse, error) {
	resp, err := c.api.PostApiV1SandboxesIdResumeWithResponse(ctx, apispec.SandboxID(sandboxID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// RefreshSandbox refreshes sandbox TTL. If request is nil, an empty body is sent.
func (c *Client) RefreshSandbox(ctx context.Context, sandboxID string, request *apispec.RefreshRequest) (*apispec.RefreshResponse, error) {
	var (
		resp *apispec.PostApiV1SandboxesIdRefreshResponse
		err  error
	)
	if request == nil {
		resp, err = c.api.PostApiV1SandboxesIdRefreshWithBodyWithResponse(
			ctx,
			apispec.SandboxID(sandboxID),
			"application/json",
			http.NoBody,
		)
	} else {
		resp, err = c.api.PostApiV1SandboxesIdRefreshWithResponse(ctx, apispec.SandboxID(sandboxID), *request)
	}
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}
