package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// Mount mounts a volume into a sandbox.
func (s *Sandbox) Mount(ctx context.Context, volumeID, mountPoint string, config *apispec.VolumeConfig) (*apispec.MountResponse, error) {
	req := apispec.MountRequest{
		MountPoint:      mountPoint,
		SandboxvolumeId: volumeID,
		VolumeConfig:    config,
	}
	resp, err := s.client.api.PostApiV1SandboxesIdSandboxvolumesMountWithResponse(ctx, apispec.SandboxID(s.ID), req)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Unmount unmounts a volume from a sandbox.
func (s *Sandbox) Unmount(ctx context.Context, volumeID string) (*apispec.SuccessUnmountedResponse, error) {
	req := apispec.UnmountRequest{
		SandboxvolumeId: volumeID,
	}
	resp, err := s.client.api.PostApiV1SandboxesIdSandboxvolumesUnmountWithResponse(ctx, apispec.SandboxID(s.ID), req)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// MountStatus returns mount status for a sandbox.
func (s *Sandbox) MountStatus(ctx context.Context) ([]apispec.MountStatus, error) {
	resp, err := s.client.api.GetApiV1SandboxesIdSandboxvolumesStatusWithResponse(ctx, apispec.SandboxID(s.ID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil && resp.JSON200.Data.Mounts != nil {
		return *resp.JSON200.Data.Mounts, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}
