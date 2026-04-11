package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// ImageBuildOption configures a server-side image build request.
type ImageBuildOption func(*apispec.ImageBuildRequest)

// WithImageBuildContextPath sets the relative build context directory inside the context volume.
func WithImageBuildContextPath(path string) ImageBuildOption {
	return func(req *apispec.ImageBuildRequest) {
		req.ContextPath = apispec.NewOptString(path)
	}
}

// WithImageBuildDockerfilePath sets the relative Dockerfile path inside the selected context directory.
func WithImageBuildDockerfilePath(path string) ImageBuildOption {
	return func(req *apispec.ImageBuildRequest) {
		req.DockerfilePath = apispec.NewOptString(path)
	}
}

// WithImageBuildCacheVolume mounts an existing SandboxVolume as the build cache.
func WithImageBuildCacheVolume(volumeID string) ImageBuildOption {
	return func(req *apispec.ImageBuildRequest) {
		req.CacheVolumeID = apispec.NewOptString(volumeID)
	}
}

// WithImageBuildPlatform requests a target platform such as linux/amd64.
func WithImageBuildPlatform(platform string) ImageBuildOption {
	return func(req *apispec.ImageBuildRequest) {
		req.Platform = apispec.NewOptString(platform)
	}
}

// WithImageBuildNoCache disables builder cache use for this build.
func WithImageBuildNoCache() ImageBuildOption {
	return func(req *apispec.ImageBuildRequest) {
		req.NoCache = apispec.NewOptBool(true)
	}
}

// WithImageBuildPull requests pulling newer base image layers during the build.
func WithImageBuildPull() ImageBuildOption {
	return func(req *apispec.ImageBuildRequest) {
		req.Pull = apispec.NewOptBool(true)
	}
}

// WithImageBuildArgs sets build args. Existing args on the request are replaced.
func WithImageBuildArgs(args map[string]string) ImageBuildOption {
	return func(req *apispec.ImageBuildRequest) {
		if len(args) == 0 {
			return
		}
		copied := make(apispec.ImageBuildRequestBuildArgs, len(args))
		for key, value := range args {
			copied[key] = value
		}
		req.BuildArgs = apispec.NewOptImageBuildRequestBuildArgs(copied)
	}
}

// StartImageBuild starts a server-side image build from an existing context volume.
func (c *Client) StartImageBuild(ctx context.Context, contextVolumeID string, opts ...ImageBuildOption) (*apispec.ImageBuildResponse, error) {
	req := apispec.ImageBuildRequest{ContextVolumeID: contextVolumeID}
	for _, opt := range opts {
		if opt != nil {
			opt(&req)
		}
	}
	return c.StartImageBuildRequest(ctx, req)
}

// StartImageBuildRequest starts a server-side image build using a fully constructed request.
func (c *Client) StartImageBuildRequest(ctx context.Context, request apispec.ImageBuildRequest) (*apispec.ImageBuildResponse, error) {
	resp, err := c.api.APIV1ImageBuildsPost(ctx, &request)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessImageBuildResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}
