package sandbox0

import (
	"context"
	"net/http"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// SandboxService provides sandbox lifecycle APIs.
type SandboxService struct {
	client *Client
}

type sandboxOptions struct {
	config *apispec.SandboxConfig
}

// SandboxOption configures sandbox creation.
type SandboxOption func(*sandboxOptions)

// WithSandboxConfig sets the sandbox configuration for creation.
func WithSandboxConfig(config apispec.SandboxConfig) SandboxOption {
	return func(opts *sandboxOptions) {
		opts.config = &config
	}
}

// Claim creates (claims) a sandbox and returns a convenience wrapper.
func (s *SandboxService) Claim(ctx context.Context, template string, opts ...SandboxOption) (*Sandbox, error) {
	options := sandboxOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	req := apispec.ClaimRequest{
		Template: &template,
		Config:   options.config,
	}

	resp, err := s.client.api.PostApiV1SandboxesWithResponse(ctx, req)
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
			client:            s.client,
			replContextByLang: map[string]string{},
		}
		sandbox.Contexts = SandboxContextService{sandbox: sandbox}
		sandbox.Files = SandboxFileService{sandbox: sandbox}
		return sandbox, nil
	}
	if resp.JSON400 != nil {
		return nil, apiErrorFromEnvelope(resp.HTTPResponse, resp.JSON400)
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Get returns sandbox details by ID.
func (s *SandboxService) Get(ctx context.Context, sandboxID string) (*apispec.Sandbox, error) {
	resp, err := s.client.api.GetApiV1SandboxesIdWithResponse(ctx, apispec.SandboxID(sandboxID))
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

// Update updates sandbox configuration.
func (s *SandboxService) Update(ctx context.Context, sandboxID string, request apispec.SandboxUpdateRequest) (*apispec.Sandbox, error) {
	resp, err := s.client.api.PatchApiV1SandboxesIdWithResponse(ctx, apispec.SandboxID(sandboxID), request)
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

// Delete terminates a sandbox.
func (s *SandboxService) Delete(ctx context.Context, sandboxID string) (*apispec.SuccessMessageResponse, error) {
	resp, err := s.client.api.DeleteApiV1SandboxesIdWithResponse(ctx, apispec.SandboxID(sandboxID))
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

// Status returns the sandbox status.
func (s *SandboxService) Status(ctx context.Context, sandboxID string) (*apispec.SandboxStatus, error) {
	resp, err := s.client.api.GetApiV1SandboxesIdStatusWithResponse(ctx, apispec.SandboxID(sandboxID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Pause suspends a sandbox.
func (s *SandboxService) Pause(ctx context.Context, sandboxID string) (*apispec.PauseSandboxResponse, error) {
	resp, err := s.client.api.PostApiV1SandboxesIdPauseWithResponse(ctx, apispec.SandboxID(sandboxID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Resume resumes a sandbox.
func (s *SandboxService) Resume(ctx context.Context, sandboxID string) (*apispec.ResumeSandboxResponse, error) {
	resp, err := s.client.api.PostApiV1SandboxesIdResumeWithResponse(ctx, apispec.SandboxID(sandboxID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Refresh refreshes sandbox TTL. If request is nil, an empty body is sent.
func (s *SandboxService) Refresh(ctx context.Context, sandboxID string, request *apispec.RefreshRequest) (*apispec.RefreshResponse, error) {
	var (
		resp *apispec.PostApiV1SandboxesIdRefreshResponse
		err  error
	)
	if request == nil {
		resp, err = s.client.api.PostApiV1SandboxesIdRefreshWithBodyWithResponse(
			ctx,
			apispec.SandboxID(sandboxID),
			"application/json",
			http.NoBody,
		)
	} else {
		resp, err = s.client.api.PostApiV1SandboxesIdRefreshWithResponse(ctx, apispec.SandboxID(sandboxID), *request)
	}
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// GetNetworkPolicy retrieves the sandbox network policy.
func (s *SandboxService) GetNetworkPolicy(ctx context.Context, sandboxID string) (*apispec.TplSandboxNetworkPolicy, error) {
	resp, err := s.client.api.GetApiV1SandboxesIdNetworkWithResponse(ctx, apispec.SandboxID(sandboxID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// UpdateNetworkPolicy updates the sandbox network policy.
func (s *SandboxService) UpdateNetworkPolicy(ctx context.Context, sandboxID string, policy apispec.TplSandboxNetworkPolicy) (*apispec.TplSandboxNetworkPolicy, error) {
	resp, err := s.client.api.PatchApiV1SandboxesIdNetworkWithResponse(ctx, apispec.SandboxID(sandboxID), policy)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// GetBandwidthPolicy retrieves the sandbox bandwidth policy.
func (s *SandboxService) GetBandwidthPolicy(ctx context.Context, sandboxID string) (*apispec.BandwidthPolicySpec, error) {
	resp, err := s.client.api.GetApiV1SandboxesIdBandwidthWithResponse(ctx, apispec.SandboxID(sandboxID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// UpdateBandwidthPolicy updates the sandbox bandwidth policy.
func (s *SandboxService) UpdateBandwidthPolicy(ctx context.Context, sandboxID string, policy apispec.BandwidthPolicySpec) (*apispec.BandwidthPolicySpec, error) {
	resp, err := s.client.api.PatchApiV1SandboxesIdBandwidthWithResponse(ctx, apispec.SandboxID(sandboxID), policy)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}
