package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// SandboxServicesResponse represents the canonical services configured on a sandbox.
type SandboxServicesResponse struct {
	SandboxID string
	Services  []apispec.SandboxAppServiceView
}

// GetServices retrieves the canonical sandbox services.
func (s *Sandbox) GetServices(ctx context.Context) (*SandboxServicesResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDServicesGet(ctx, apispec.APIV1SandboxesIDServicesGetParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxServicesResponse:
		return sandboxServicesResponseFromAPI(response)
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// UpdateServices replaces the canonical sandbox services.
func (s *Sandbox) UpdateServices(ctx context.Context, services []apispec.SandboxAppService) (*SandboxServicesResponse, error) {
	req := apispec.SandboxServicesUpdateRequest{
		Services: normalizeSandboxServices(services),
	}
	resp, err := s.client.api.APIV1SandboxesIDServicesPut(ctx, &req, apispec.APIV1SandboxesIDServicesPutParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxServicesResponse:
		return sandboxServicesResponseFromAPI(response)
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ClearServices removes all canonical sandbox services.
func (s *Sandbox) ClearServices(ctx context.Context) (*SandboxServicesResponse, error) {
	return s.UpdateServices(ctx, []apispec.SandboxAppService{})
}

func sandboxServicesResponseFromAPI(resp *apispec.SuccessSandboxServicesResponse) (*SandboxServicesResponse, error) {
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &SandboxServicesResponse{
		SandboxID: data.SandboxID,
		Services:  append([]apispec.SandboxAppServiceView(nil), data.Services...),
	}, nil
}

func normalizeSandboxServices(services []apispec.SandboxAppService) []apispec.SandboxAppService {
	if services == nil {
		return []apispec.SandboxAppService{}
	}
	return services
}
