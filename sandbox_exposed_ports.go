package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// ExposedPort represents a publicly accessible port configuration.
type ExposedPort struct {
	Port      int32
	Resume    bool
	PublicURL string // The full public URL to access this exposed port (readOnly)
}

// ExposedPortsResponse contains the response from exposed ports operations.
type ExposedPortsResponse struct {
	Ports          []ExposedPort
	ExposureDomain string // The base exposure domain (e.g., "aws-us-east-1.sandbox0.app")
}

// GetExposedPorts retrieves all exposed ports for the sandbox.
func (s *Sandbox) GetExposedPorts(ctx context.Context) (*ExposedPortsResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDExposedPortsGet(ctx, apispec.APIV1SandboxesIDExposedPortsGetParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessExposedPortsResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		result := &ExposedPortsResponse{
			Ports: make([]ExposedPort, len(data.ExposedPorts)),
		}
		for i, p := range data.ExposedPorts {
			result.Ports[i] = ExposedPort{
				Port:   p.Port,
				Resume: p.Resume,
			}
			if publicURL, ok := p.PublicURL.Get(); ok {
				result.Ports[i].PublicURL = publicURL
			}
		}
		if domain, ok := data.ExposureDomain.Get(); ok {
			result.ExposureDomain = domain
		}
		return result, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// UpdateExposedPorts replaces all exposed ports for the sandbox.
// This is useful when you want to set multiple ports at once.
func (s *Sandbox) UpdateExposedPorts(ctx context.Context, ports []ExposedPort) (*ExposedPortsResponse, error) {
	reqPorts := make([]apispec.ExposedPortConfig, len(ports))
	for i, p := range ports {
		reqPorts[i] = apispec.ExposedPortConfig{
			Port:   p.Port,
			Resume: p.Resume,
		}
	}
	req := &apispec.UpdateExposedPortsRequest{
		Ports: reqPorts,
	}
	resp, err := s.client.api.APIV1SandboxesIDExposedPortsPut(ctx, req, apispec.APIV1SandboxesIDExposedPortsPutParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessExposedPortsResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		result := &ExposedPortsResponse{
			Ports: make([]ExposedPort, len(data.ExposedPorts)),
		}
		for i, p := range data.ExposedPorts {
			result.Ports[i] = ExposedPort{
				Port:   p.Port,
				Resume: p.Resume,
			}
			if publicURL, ok := p.PublicURL.Get(); ok {
				result.Ports[i].PublicURL = publicURL
			}
		}
		if domain, ok := data.ExposureDomain.Get(); ok {
			result.ExposureDomain = domain
		}
		return result, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ExposePort adds or updates a single exposed port.
// This is a convenience method that fetches existing ports, updates the specified port, and saves.
func (s *Sandbox) ExposePort(ctx context.Context, port int32, resume bool) (*ExposedPortsResponse, error) {
	// Get current ports
	current, err := s.GetExposedPorts(ctx)
	if err != nil {
		return nil, err
	}

	// Update or add the port
	found := false
	for i, p := range current.Ports {
		if p.Port == port {
			current.Ports[i].Resume = resume
			found = true
			break
		}
	}
	if !found {
		current.Ports = append(current.Ports, ExposedPort{Port: port, Resume: resume})
	}

	// Save updated ports
	return s.UpdateExposedPorts(ctx, current.Ports)
}

// UnexposePort removes a specific exposed port.
// Returns the remaining exposed ports.
func (s *Sandbox) UnexposePort(ctx context.Context, port int32) (*ExposedPortsResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDExposedPortsPortDelete(ctx, apispec.APIV1SandboxesIDExposedPortsPortDeleteParams{
		ID:   s.ID,
		Port: port,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessExposedPortsResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		result := &ExposedPortsResponse{
			Ports: make([]ExposedPort, len(data.ExposedPorts)),
		}
		for i, p := range data.ExposedPorts {
			result.Ports[i] = ExposedPort{
				Port:   p.Port,
				Resume: p.Resume,
			}
			if publicURL, ok := p.PublicURL.Get(); ok {
				result.Ports[i].PublicURL = publicURL
			}
		}
		if domain, ok := data.ExposureDomain.Get(); ok {
			result.ExposureDomain = domain
		}
		return result, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ClearExposedPorts removes all exposed ports for the sandbox.
func (s *Sandbox) ClearExposedPorts(ctx context.Context) error {
	_, err := s.client.api.APIV1SandboxesIDExposedPortsDelete(ctx, apispec.APIV1SandboxesIDExposedPortsDeleteParams{ID: s.ID})
	return err
}
