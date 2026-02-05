package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// GetNetworkPolicy retrieves the sandbox network policy.
func (s *Sandbox) GetNetworkPolicy(ctx context.Context) (*apispec.TplSandboxNetworkPolicy, error) {
	resp, err := s.client.api.APIV1SandboxesIDNetworkGet(ctx, apispec.APIV1SandboxesIDNetworkGetParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxNetworkPolicyResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// UpdateNetworkPolicy updates the sandbox network policy.
func (s *Sandbox) UpdateNetworkPolicy(ctx context.Context, policy apispec.TplSandboxNetworkPolicy) (*apispec.TplSandboxNetworkPolicy, error) {
	resp, err := s.client.api.APIV1SandboxesIDNetworkPatch(ctx, &policy, apispec.APIV1SandboxesIDNetworkPatchParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}

// GetBandwidthPolicy retrieves the sandbox bandwidth policy.
func (s *Sandbox) GetBandwidthPolicy(ctx context.Context) (*apispec.BandwidthPolicySpec, error) {
	resp, err := s.client.api.APIV1SandboxesIDBandwidthGet(ctx, apispec.APIV1SandboxesIDBandwidthGetParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}

// UpdateBandwidthPolicy updates the sandbox bandwidth policy.
func (s *Sandbox) UpdateBandwidthPolicy(ctx context.Context, policy apispec.BandwidthPolicySpec) (*apispec.BandwidthPolicySpec, error) {
	resp, err := s.client.api.APIV1SandboxesIDBandwidthPatch(ctx, &policy, apispec.APIV1SandboxesIDBandwidthPatchParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}
