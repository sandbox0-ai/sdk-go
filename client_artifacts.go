package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// CreateArtifactOption configures artifact creation helpers.
type CreateArtifactOption func(*apispec.CreateArtifactRequest)

// WithArtifactName sets a human-readable artifact name.
func WithArtifactName(name string) CreateArtifactOption {
	return func(req *apispec.CreateArtifactRequest) {
		req.Name = apispec.NewOptString(name)
	}
}

// WithArtifactKind sets an application-defined artifact kind.
func WithArtifactKind(kind string) CreateArtifactOption {
	return func(req *apispec.CreateArtifactRequest) {
		req.Kind = apispec.NewOptString(kind)
	}
}

// WithArtifactMediaType sets the artifact media type.
func WithArtifactMediaType(mediaType string) CreateArtifactOption {
	return func(req *apispec.CreateArtifactRequest) {
		req.MediaType = apispec.NewOptString(mediaType)
	}
}

// WithArtifactDigest records an optional caller-supplied content digest.
func WithArtifactDigest(digest string) CreateArtifactOption {
	return func(req *apispec.CreateArtifactRequest) {
		req.Digest = apispec.NewOptString(digest)
	}
}

// WithArtifactMetadata sets caller-defined artifact metadata.
func WithArtifactMetadata(metadata apispec.CreateArtifactRequestMetadata) CreateArtifactOption {
	return func(req *apispec.CreateArtifactRequest) {
		req.Metadata = apispec.NewOptCreateArtifactRequestMetadata(metadata)
	}
}

// ArtifactSourceSandboxVolume builds an artifact source from a SandboxVolume.
func ArtifactSourceSandboxVolume(volumeID string) apispec.CreateArtifactSource {
	return apispec.CreateArtifactSource{
		Type:            apispec.CreateArtifactSourceTypeSandboxVolume,
		SandboxvolumeID: volumeID,
	}
}

// CreateArtifact creates an artifact from a source.
func (c *Client) CreateArtifact(ctx context.Context, request apispec.CreateArtifactRequest) (*apispec.Artifact, error) {
	resp, err := c.api.APIV1ArtifactsPost(ctx, &request)
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}

// CreateArtifactFromVolume creates an artifact snapshot from a SandboxVolume.
func (c *Client) CreateArtifactFromVolume(ctx context.Context, volumeID string, opts ...CreateArtifactOption) (*apispec.Artifact, error) {
	req := apispec.CreateArtifactRequest{
		Source: ArtifactSourceSandboxVolume(volumeID),
	}
	for _, opt := range opts {
		opt(&req)
	}
	return c.CreateArtifact(ctx, req)
}

// ListArtifacts lists artifacts for the current team.
func (c *Client) ListArtifacts(ctx context.Context) ([]apispec.Artifact, error) {
	resp, err := c.api.APIV1ArtifactsGet(ctx)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return data.Artifacts, nil
}

// GetArtifact retrieves an artifact by ID.
func (c *Client) GetArtifact(ctx context.Context, artifactID string) (*apispec.Artifact, error) {
	resp, err := c.api.APIV1ArtifactsIDGet(ctx, apispec.APIV1ArtifactsIDGetParams{ID: artifactID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessArtifactResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteArtifact deletes an artifact.
func (c *Client) DeleteArtifact(ctx context.Context, artifactID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := c.api.APIV1ArtifactsIDDelete(ctx, apispec.APIV1ArtifactsIDDeleteParams{ID: artifactID})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// MaterializeArtifactVolume creates a SandboxVolume from artifact contents.
func (c *Client) MaterializeArtifactVolume(ctx context.Context, artifactID string, request *apispec.CreateArtifactVolumeRequest) (*apispec.SandboxVolume, error) {
	var req apispec.OptCreateArtifactVolumeRequest
	if request != nil {
		req = apispec.NewOptCreateArtifactVolumeRequest(*request)
	}
	resp, err := c.api.APIV1ArtifactsIDVolumePost(ctx, req, apispec.APIV1ArtifactsIDVolumePostParams{ID: artifactID})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}
