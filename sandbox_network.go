package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// GetNetworkPolicy retrieves the sandbox network policy.
func (s *Sandbox) GetNetworkPolicy(ctx context.Context) (*apispec.SandboxNetworkPolicy, error) {
	resp, err := s.client.api.APIV1SandboxesIDNetworkGet(ctx, apispec.APIV1SandboxesIDNetworkGetParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxNetworkPolicyResponse:
		data := response.Data
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// UpdateNetworkPolicy updates the sandbox network policy.
func (s *Sandbox) UpdateNetworkPolicy(ctx context.Context, policy apispec.SandboxNetworkPolicy) (*apispec.SandboxNetworkPolicy, error) {
	resp, err := s.client.api.APIV1SandboxesIDNetworkPut(ctx, &policy, apispec.APIV1SandboxesIDNetworkPutParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	response, err := expectResponse[apispec.SuccessSandboxNetworkPolicyResponse](resp)
	if err != nil {
		return nil, err
	}
	data := response.Data
	return &data, nil
}
