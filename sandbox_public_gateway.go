package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// PublicGatewayResponse represents the current public gateway policy for a sandbox.
type PublicGatewayResponse struct {
	SandboxID      string
	PublicGateway  apispec.PublicGatewayConfig
	ExposureDomain string
}

// GetPublicGateway retrieves the sandbox public gateway policy.
func (s *Sandbox) GetPublicGateway(ctx context.Context) (*PublicGatewayResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDPublicGatewayGet(ctx, apispec.APIV1SandboxesIDPublicGatewayGetParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessPublicGatewayResponse:
		return publicGatewayResponseFromAPI(response)
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// UpdatePublicGateway replaces the sandbox public gateway policy.
func (s *Sandbox) UpdatePublicGateway(ctx context.Context, config apispec.PublicGatewayConfig) (*PublicGatewayResponse, error) {
	config = normalizePublicGatewayConfig(config)
	resp, err := s.client.api.APIV1SandboxesIDPublicGatewayPut(ctx, &config, apispec.APIV1SandboxesIDPublicGatewayPutParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessPublicGatewayResponse:
		return publicGatewayResponseFromAPI(response)
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ClearPublicGateway disables request-level public gateway enforcement.
func (s *Sandbox) ClearPublicGateway(ctx context.Context) (*PublicGatewayResponse, error) {
	return s.UpdatePublicGateway(ctx, apispec.PublicGatewayConfig{
		Enabled: false,
		Routes:  []apispec.PublicGatewayRoute{},
	})
}

func publicGatewayResponseFromAPI(resp *apispec.SuccessPublicGatewayResponse) (*PublicGatewayResponse, error) {
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &PublicGatewayResponse{
		SandboxID:      data.SandboxID,
		PublicGateway:  data.PublicGateway,
		ExposureDomain: data.ExposureDomain.Or(""),
	}, nil
}

func normalizePublicGatewayConfig(config apispec.PublicGatewayConfig) apispec.PublicGatewayConfig {
	if config.Routes == nil {
		config.Routes = []apispec.PublicGatewayRoute{}
	}
	return config
}

func normalizeClaimRequest(req apispec.ClaimRequest) apispec.ClaimRequest {
	config, ok := req.Config.Get()
	if !ok {
		return req
	}
	publicGateway, ok := config.PublicGateway.Get()
	if !ok {
		return req
	}
	config.PublicGateway = apispec.NewOptPublicGatewayConfig(normalizePublicGatewayConfig(publicGateway))
	req.Config = apispec.NewOptSandboxConfig(config)
	return req
}

func normalizeSandboxUpdateRequest(req apispec.SandboxUpdateRequest) apispec.SandboxUpdateRequest {
	config, ok := req.Config.Get()
	if !ok {
		return req
	}
	publicGateway, ok := config.PublicGateway.Get()
	if !ok {
		return req
	}
	config.PublicGateway = apispec.NewOptPublicGatewayConfig(normalizePublicGatewayConfig(publicGateway))
	req.Config = apispec.NewOptSandboxUpdateConfig(config)
	return req
}
