package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// VolumeService provides sandbox volume APIs.
type VolumeService struct {
	client *Client
}

// Create creates a sandbox volume.
func (s *VolumeService) Create(ctx context.Context, request apispec.CreateSandboxVolumeRequest) (*apispec.SandboxVolume, error) {
	resp, err := s.client.api.PostApiV1SandboxvolumesWithResponse(ctx, request)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 != nil && resp.JSON201.Data != nil {
		return resp.JSON201.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// List lists sandbox volumes.
func (s *VolumeService) List(ctx context.Context) ([]apispec.SandboxVolume, error) {
	resp, err := s.client.api.GetApiV1SandboxvolumesWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return *resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Get retrieves a sandbox volume.
func (s *VolumeService) Get(ctx context.Context, volumeID string) (*apispec.SandboxVolume, error) {
	resp, err := s.client.api.GetApiV1SandboxvolumesIdWithResponse(ctx, apispec.SandboxVolumeID(volumeID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	if resp.JSON404 != nil {
		return nil, apiErrorFromEnvelope(resp.HTTPResponse, resp.JSON404)
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Delete deletes a sandbox volume.
func (s *VolumeService) Delete(ctx context.Context, volumeID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := s.client.api.DeleteApiV1SandboxvolumesIdWithResponse(ctx, apispec.SandboxVolumeID(volumeID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	if resp.JSON409 != nil {
		return nil, apiErrorFromEnvelope(resp.HTTPResponse, resp.JSON409)
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Mount mounts a volume into a sandbox.
func (s *VolumeService) Mount(ctx context.Context, sandboxID, volumeID, mountPoint string, config *apispec.VolumeConfig) (*apispec.MountResponse, error) {
	req := apispec.MountRequest{
		MountPoint:      mountPoint,
		SandboxvolumeId: volumeID,
		VolumeConfig:    config,
	}
	resp, err := s.client.api.PostApiV1SandboxesIdSandboxvolumesMountWithResponse(ctx, apispec.SandboxID(sandboxID), req)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Unmount unmounts a volume from a sandbox.
func (s *VolumeService) Unmount(ctx context.Context, sandboxID, volumeID string) (*apispec.SuccessUnmountedResponse, error) {
	resp, err := s.client.api.PostApiV1SandboxesIdSandboxvolumesUnmountWithResponse(ctx, apispec.SandboxID(sandboxID), apispec.UnmountRequest{
		SandboxvolumeId: volumeID,
	})
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// MountStatus returns mount status for a sandbox.
func (s *VolumeService) MountStatus(ctx context.Context, sandboxID string) ([]apispec.MountStatus, error) {
	resp, err := s.client.api.GetApiV1SandboxesIdSandboxvolumesStatusWithResponse(ctx, apispec.SandboxID(sandboxID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil && resp.JSON200.Data.Mounts != nil {
		return *resp.JSON200.Data.Mounts, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// CreateSnapshot creates a snapshot for a volume.
func (s *VolumeService) CreateSnapshot(ctx context.Context, volumeID string, request apispec.CreateSnapshotRequest) (*apispec.Snapshot, error) {
	resp, err := s.client.api.PostApiV1SandboxvolumesIdSnapshotsWithResponse(ctx, apispec.SandboxVolumeID(volumeID), request)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 != nil && resp.JSON201.Data != nil {
		return resp.JSON201.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// ListSnapshots lists snapshots for a volume.
func (s *VolumeService) ListSnapshots(ctx context.Context, volumeID string) ([]apispec.Snapshot, error) {
	resp, err := s.client.api.GetApiV1SandboxvolumesIdSnapshotsWithResponse(ctx, apispec.SandboxVolumeID(volumeID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return *resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// GetSnapshot gets a snapshot.
func (s *VolumeService) GetSnapshot(ctx context.Context, volumeID, snapshotID string) (*apispec.Snapshot, error) {
	resp, err := s.client.api.GetApiV1SandboxvolumesIdSnapshotsSnapshotIdWithResponse(ctx, apispec.SandboxVolumeID(volumeID), apispec.SnapshotID(snapshotID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	if resp.JSON404 != nil {
		return nil, apiErrorFromEnvelope(resp.HTTPResponse, resp.JSON404)
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// DeleteSnapshot deletes a snapshot.
func (s *VolumeService) DeleteSnapshot(ctx context.Context, volumeID, snapshotID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := s.client.api.DeleteApiV1SandboxvolumesIdSnapshotsSnapshotIdWithResponse(ctx, apispec.SandboxVolumeID(volumeID), apispec.SnapshotID(snapshotID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// RestoreSnapshot restores a snapshot.
func (s *VolumeService) RestoreSnapshot(ctx context.Context, volumeID, snapshotID string) (*apispec.SuccessRestoreResponse, error) {
	resp, err := s.client.api.PostApiV1SandboxvolumesIdSnapshotsSnapshotIdRestoreWithResponse(ctx, apispec.SandboxVolumeID(volumeID), apispec.SnapshotID(snapshotID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}
