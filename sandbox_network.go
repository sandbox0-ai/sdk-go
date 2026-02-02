package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// GetNetworkPolicy retrieves the sandbox network policy.
func (s *Sandbox) GetNetworkPolicy(ctx context.Context) (*apispec.TplSandboxNetworkPolicy, error) {
	resp, err := s.client.api.GetApiV1SandboxesIdNetworkWithResponse(ctx, apispec.SandboxID(s.ID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// UpdateNetworkPolicy updates the sandbox network policy.
func (s *Sandbox) UpdateNetworkPolicy(ctx context.Context, policy apispec.TplSandboxNetworkPolicy) (*apispec.TplSandboxNetworkPolicy, error) {
	resp, err := s.client.api.PatchApiV1SandboxesIdNetworkWithResponse(ctx, apispec.SandboxID(s.ID), policy)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// GetBandwidthPolicy retrieves the sandbox bandwidth policy.
func (s *Sandbox) GetBandwidthPolicy(ctx context.Context) (*apispec.BandwidthPolicySpec, error) {
	resp, err := s.client.api.GetApiV1SandboxesIdBandwidthWithResponse(ctx, apispec.SandboxID(s.ID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// UpdateBandwidthPolicy updates the sandbox bandwidth policy.
func (s *Sandbox) UpdateBandwidthPolicy(ctx context.Context, policy apispec.BandwidthPolicySpec) (*apispec.BandwidthPolicySpec, error) {
	resp, err := s.client.api.PatchApiV1SandboxesIdBandwidthWithResponse(ctx, apispec.SandboxID(s.ID), policy)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}
