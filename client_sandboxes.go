package sandbox0

import (
	"context"

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
		config.TTL = apispec.NewOptInt32(ttlSec)
	}
}

// WithSandboxHardTTL sets the hard TTL (seconds) for created sandboxes.
func WithSandboxHardTTL(ttlSec int32) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		config.HardTTL = apispec.NewOptInt32(ttlSec)
	}
}

// WithSandboxWebhook configures webhook delivery for sandbox events.
func WithSandboxWebhook(url, secret string) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		webhook := apispec.WebhookConfig{}
		if existing, ok := config.Webhook.Get(); ok {
			webhook = existing
		}
		webhook.URL = apispec.NewOptString(url)
		webhook.Secret = apispec.NewOptString(secret)
		config.Webhook = apispec.NewOptWebhookConfig(webhook)
	}
}

// WithSandboxWebhookWatchDir sets the webhook watch directory (file events).
func WithSandboxWebhookWatchDir(watchDir string) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		webhook := apispec.WebhookConfig{}
		if existing, ok := config.Webhook.Get(); ok {
			webhook = existing
		}
		webhook.WatchDir = apispec.NewOptString(watchDir)
		config.Webhook = apispec.NewOptWebhookConfig(webhook)
	}
}

// WithSandboxNetworkPolicy sets the sandbox network policy at claim time.
func WithSandboxNetworkPolicy(policy apispec.TplSandboxNetworkPolicy) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		config.Network = apispec.NewOptTplSandboxNetworkPolicy(policy)
	}
}

// ClaimSandbox creates (claims) a sandbox and returns a convenience wrapper.
func (c *Client) ClaimSandbox(ctx context.Context, template string, opts ...SandboxOption) (*Sandbox, error) {
	options := sandboxOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	req := apispec.ClaimRequest{
		Template: apispec.NewOptString(template),
	}
	if options.config != nil {
		req.Config = apispec.NewOptSandboxConfig(*options.config)
	}

	resp, err := c.api.APIV1SandboxesPost(ctx, &req)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessClaimResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		var clusterID *string
		if value, ok := data.ClusterID.Get(); ok {
			clusterID = &value
		}
		sandbox := &Sandbox{
			ID:                data.SandboxID,
			Template:          data.Template,
			ClusterID:         clusterID,
			PodName:           data.PodName,
			Status:            data.Status,
			client:            c,
			replContextByLang: map[string]string{},
		}
		return sandbox, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// GetSandbox returns sandbox details by ID.
func (c *Client) GetSandbox(ctx context.Context, sandboxID string) (*apispec.Sandbox, error) {
	resp, err := c.api.APIV1SandboxesIDGet(ctx, apispec.APIV1SandboxesIDGetParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// UpdateSandbox updates sandbox configuration.
func (c *Client) UpdateSandbox(ctx context.Context, sandboxID string, request apispec.SandboxUpdateRequest) (*apispec.Sandbox, error) {
	resp, err := c.api.APIV1SandboxesIDPatch(ctx, &request, apispec.APIV1SandboxesIDPatchParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteSandbox terminates a sandbox.
func (c *Client) DeleteSandbox(ctx context.Context, sandboxID string) (*apispec.SuccessMessageResponse, error) {
	resp, err := c.api.APIV1SandboxesIDDelete(ctx, apispec.APIV1SandboxesIDDeleteParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessMessageResponse:
		return response, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// StatusSandbox returns the sandbox status.
func (c *Client) StatusSandbox(ctx context.Context, sandboxID string) (*apispec.SandboxStatus, error) {
	resp, err := c.api.APIV1SandboxesIDStatusGet(ctx, apispec.APIV1SandboxesIDStatusGetParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxStatusResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	case *apispec.ErrorEnvelope:
		return nil, apiErrorFromResponse(response)
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// PauseSandbox suspends a sandbox.
func (c *Client) PauseSandbox(ctx context.Context, sandboxID string) (*apispec.PauseSandboxResponse, error) {
	resp, err := c.api.APIV1SandboxesIDPausePost(ctx, apispec.APIV1SandboxesIDPausePostParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessPauseSandboxResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	case *apispec.ErrorEnvelope:
		return nil, apiErrorFromResponse(response)
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ResumeSandbox resumes a sandbox.
func (c *Client) ResumeSandbox(ctx context.Context, sandboxID string) (*apispec.ResumeSandboxResponse, error) {
	resp, err := c.api.APIV1SandboxesIDResumePost(ctx, apispec.APIV1SandboxesIDResumePostParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessResumeSandboxResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	case *apispec.ErrorEnvelope:
		return nil, apiErrorFromResponse(response)
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// RefreshSandbox refreshes sandbox TTL. If request is nil, an empty body is sent.
func (c *Client) RefreshSandbox(ctx context.Context, sandboxID string, request *apispec.RefreshRequest) (*apispec.RefreshResponse, error) {
	var (
		resp apispec.APIV1SandboxesIDRefreshPostRes
		err  error
	)
	if request == nil {
		resp, err = c.api.APIV1SandboxesIDRefreshPost(ctx, apispec.OptRefreshRequest{}, apispec.APIV1SandboxesIDRefreshPostParams{ID: sandboxID})
	} else {
		resp, err = c.api.APIV1SandboxesIDRefreshPost(ctx, apispec.NewOptRefreshRequest(*request), apispec.APIV1SandboxesIDRefreshPostParams{ID: sandboxID})
	}
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessRefreshResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	case *apispec.ErrorEnvelope:
		return nil, apiErrorFromResponse(response)
	default:
		if err := apiErrorFromResponse(response); err != nil {
			return nil, err
		}
		return nil, unexpectedResponseError(response)
	}
}
