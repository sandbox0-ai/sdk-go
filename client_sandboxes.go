package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

type sandboxOptions struct {
	config     *apispec.SandboxConfig
	snapshotID string
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

// WithSandboxSnapshotID initializes the new sandbox root filesystem from a rootfs snapshot.
func WithSandboxSnapshotID(snapshotID string) SandboxOption {
	return func(opts *sandboxOptions) {
		opts.snapshotID = snapshotID
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

// WithSandboxMemory sets the sandbox memory limit, such as "512Mi" or "2Gi".
// Sandbox0 derives CPU from the platform memory-per-CPU ratio.
func WithSandboxMemory(memory string) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		config.Resources = apispec.NewOptSandboxResourceConfig(apispec.SandboxResourceConfig{
			Memory: apispec.NewOptString(memory),
		})
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
func WithSandboxNetworkPolicy(policy apispec.SandboxNetworkPolicy) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		config.Network = apispec.NewOptSandboxNetworkPolicy(policy)
	}
}

// WithSandboxServices sets sandbox services at claim time.
func WithSandboxServices(services []apispec.SandboxAppService) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		config.Services = normalizeSandboxServices(services)
	}
}

// WithSandboxAutoResume controls whether paused sandbox auto resumes on access.
// Default is false when unset.
func WithSandboxAutoResume(enabled bool) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		config.AutoResume = apispec.NewOptBool(enabled)
	}
}

// WithSandboxEnvVars sets environment variables for created sandboxes.
func WithSandboxEnvVars(envVars map[string]string) SandboxOption {
	return func(opts *sandboxOptions) {
		config := ensureSandboxConfig(opts)
		config.EnvVars = apispec.NewOptSandboxConfigEnvVars(envVars)
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
	if options.snapshotID != "" {
		req.SnapshotID = apispec.NewOptString(options.snapshotID)
	}

	return c.ClaimSandboxRequest(ctx, req)
}

// ClaimSandboxRequest claims a sandbox using a fully constructed request.
func (c *Client) ClaimSandboxRequest(ctx context.Context, req apispec.ClaimRequest) (*Sandbox, error) {
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
			Status:            string(data.Status),
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
	resp, err := c.api.APIV1SandboxesIDPut(ctx, &request, apispec.APIV1SandboxesIDPutParams{ID: sandboxID})
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

// UpdateSandboxMemory updates the sandbox memory limit. Sandbox0 derives CPU
// from the platform memory-per-CPU ratio.
func (c *Client) UpdateSandboxMemory(ctx context.Context, sandboxID, memory string) (*apispec.Sandbox, error) {
	return c.UpdateSandbox(ctx, sandboxID, apispec.SandboxUpdateRequest{
		Config: apispec.NewOptSandboxUpdateConfig(apispec.SandboxUpdateConfig{
			Resources: apispec.NewOptSandboxResourceConfig(apispec.SandboxResourceConfig{
				Memory: apispec.NewOptString(memory),
			}),
		}),
	})
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
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// RefreshSandbox refreshes sandbox TTL. If request is nil, an empty body is sent.
func (c *Client) RefreshSandbox(ctx context.Context, sandboxID string, request *apispec.SandboxRefreshRequest) (*apispec.RefreshResponse, error) {
	var (
		resp apispec.APIV1SandboxesIDRefreshPostRes
		err  error
	)
	if request == nil {
		resp, err = c.api.APIV1SandboxesIDRefreshPost(ctx, apispec.OptSandboxRefreshRequest{}, apispec.APIV1SandboxesIDRefreshPostParams{ID: sandboxID})
	} else {
		resp, err = c.api.APIV1SandboxesIDRefreshPost(ctx, apispec.NewOptSandboxRefreshRequest(*request), apispec.APIV1SandboxesIDRefreshPostParams{ID: sandboxID})
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

// CreateSandboxRootFSSnapshot creates a root filesystem snapshot for a paused sandbox.
func (c *Client) CreateSandboxRootFSSnapshot(ctx context.Context, sandboxID string, request *apispec.CreateSandboxRootFSSnapshotRequest) (*apispec.SandboxRootFSSnapshot, error) {
	var req apispec.OptCreateSandboxRootFSSnapshotRequest
	if request != nil {
		req = apispec.NewOptCreateSandboxRootFSSnapshotRequest(*request)
	}
	resp, err := c.api.APIV1SandboxesIDSnapshotsPost(ctx, req, apispec.APIV1SandboxesIDSnapshotsPostParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxRootFSSnapshotResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ListSandboxRootFSSnapshots lists root filesystem snapshots for a sandbox.
func (c *Client) ListSandboxRootFSSnapshots(ctx context.Context, sandboxID string) (*apispec.SandboxRootFSSnapshotList, error) {
	resp, err := c.api.APIV1SandboxesIDSnapshotsGet(ctx, apispec.APIV1SandboxesIDSnapshotsGetParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxRootFSSnapshotListResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// GetSandboxRootFSSnapshot returns a root filesystem snapshot by ID.
func (c *Client) GetSandboxRootFSSnapshot(ctx context.Context, snapshotID string) (*apispec.SandboxRootFSSnapshot, error) {
	resp, err := c.api.APIV1SandboxRootfsSnapshotsSnapshotIDGet(ctx, apispec.APIV1SandboxRootfsSnapshotsSnapshotIDGetParams{SnapshotID: snapshotID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxRootFSSnapshotResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteSandboxRootFSSnapshot deletes a root filesystem snapshot by ID.
func (c *Client) DeleteSandboxRootFSSnapshot(ctx context.Context, snapshotID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := c.api.APIV1SandboxRootfsSnapshotsSnapshotIDDelete(ctx, apispec.APIV1SandboxRootfsSnapshotsSnapshotIDDeleteParams{SnapshotID: snapshotID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessDeletedResponse:
		return response, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// RestoreSandboxRootFS restores a paused sandbox root filesystem from a rootfs snapshot.
func (c *Client) RestoreSandboxRootFS(ctx context.Context, sandboxID string, request apispec.RestoreSandboxRootFSRequest) (*apispec.RestoreSandboxRootFSResponse, error) {
	resp, err := c.api.APIV1SandboxesIDRootfsRestorePost(ctx, &request, apispec.APIV1SandboxesIDRootfsRestorePostParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessRestoreSandboxRootFSResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ForkSandbox creates a paused sandbox fork from a paused source sandbox root filesystem.
func (c *Client) ForkSandbox(ctx context.Context, sandboxID string, request *apispec.ForkSandboxRequest) (*apispec.ForkSandboxResponse, error) {
	body := apispec.NewOptForkSandboxRequest(apispec.ForkSandboxRequest{})
	if request != nil {
		body = apispec.NewOptForkSandboxRequest(*request)
	}
	resp, err := c.api.APIV1SandboxesIDForkPost(ctx, body, apispec.APIV1SandboxesIDForkPostParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessForkSandboxResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ListSandboxesOptions configures the list sandboxes request.
type ListSandboxesOptions struct {
	Status     string
	TemplateID string
	Paused     *bool
	Limit      *int
	Offset     *int
}

// ListSandboxesResponse represents the response from listing sandboxes.
type ListSandboxesResponse struct {
	Sandboxes []apispec.SandboxSummary
	Count     int
	HasMore   bool
}

// ListSandboxes lists all sandboxes for the authenticated team.
func (c *Client) ListSandboxes(ctx context.Context, opts *ListSandboxesOptions) (*ListSandboxesResponse, error) {
	params := apispec.APIV1SandboxesGetParams{}
	if opts != nil {
		if opts.Status != "" {
			params.Status = apispec.NewOptSandboxLifecycleStatus(apispec.SandboxLifecycleStatus(opts.Status))
		}
		if opts.TemplateID != "" {
			params.TemplateID = apispec.NewOptString(opts.TemplateID)
		}
		if opts.Paused != nil {
			params.Paused = apispec.NewOptBool(*opts.Paused)
		}
		if opts.Limit != nil {
			params.Limit = apispec.NewOptInt(*opts.Limit)
		}
		if opts.Offset != nil {
			params.Offset = apispec.NewOptInt(*opts.Offset)
		}
	}

	resp, err := c.api.APIV1SandboxesGet(ctx, params)
	if err != nil {
		return nil, err
	}

	switch response := resp.(type) {
	case *apispec.SuccessSandboxListResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &ListSandboxesResponse{
			Sandboxes: data.Sandboxes,
			Count:     data.Count,
			HasMore:   data.HasMore,
		}, nil
	case *apispec.ErrorEnvelope:
		return nil, apiErrorFromResponse(response)
	default:
		return nil, apiErrorFromResponse(response)
	}
}
