package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// Mount mounts a volume into a sandbox.
func (s *Sandbox) Mount(ctx context.Context, volumeID, mountPoint string, config *apispec.VolumeConfig) (*apispec.MountResponse, error) {
	req := apispec.MountRequest{
		MountPoint:      mountPoint,
		SandboxvolumeID: volumeID,
	}
	if config != nil {
		req.VolumeConfig = apispec.NewOptVolumeConfig(*config)
	}
	resp, err := s.client.api.APIV1SandboxesIDSandboxvolumesMountPost(ctx, &req, apispec.APIV1SandboxesIDSandboxvolumesMountPostParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}

// Unmount unmounts a volume from a sandbox.
func (s *Sandbox) Unmount(ctx context.Context, volumeID, mountSessionID string) (*apispec.SuccessUnmountedResponse, error) {
	req := apispec.UnmountRequest{
		SandboxvolumeID: volumeID,
		MountSessionID:  mountSessionID,
	}
	resp, err := s.client.api.APIV1SandboxesIDSandboxvolumesUnmountPost(ctx, &req, apispec.APIV1SandboxesIDSandboxvolumesUnmountPostParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// MountStatus returns mount status for a sandbox.
func (s *Sandbox) MountStatus(ctx context.Context) ([]apispec.MountStatus, error) {
	resp, err := s.client.api.APIV1SandboxesIDSandboxvolumesStatusGet(ctx, apispec.APIV1SandboxesIDSandboxvolumesStatusGetParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return data.Mounts, nil
}
