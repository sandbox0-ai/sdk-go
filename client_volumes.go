package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// CreateVolume creates a sandbox volume.
func (c *Client) CreateVolume(ctx context.Context, request apispec.CreateSandboxVolumeRequest) (*apispec.SandboxVolume, error) {
	resp, err := c.api.PostApiV1SandboxvolumesWithResponse(ctx, request)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 != nil && resp.JSON201.Data != nil {
		return resp.JSON201.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// ListVolume lists sandbox volumes.
func (c *Client) ListVolume(ctx context.Context) ([]apispec.SandboxVolume, error) {
	resp, err := c.api.GetApiV1SandboxvolumesWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return *resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// GetVolume retrieves a sandbox volume.
func (c *Client) GetVolume(ctx context.Context, volumeID string) (*apispec.SandboxVolume, error) {
	resp, err := c.api.GetApiV1SandboxvolumesIdWithResponse(ctx, apispec.SandboxVolumeID(volumeID))
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

// DeleteVolume deletes a sandbox volume.
func (c *Client) DeleteVolume(ctx context.Context, volumeID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := c.api.DeleteApiV1SandboxvolumesIdWithResponse(ctx, apispec.SandboxVolumeID(volumeID))
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

// CreateVolumeSnapshot creates a snapshot for a volume.
func (c *Client) CreateVolumeSnapshot(ctx context.Context, volumeID string, request apispec.CreateSnapshotRequest) (*apispec.Snapshot, error) {
	resp, err := c.api.PostApiV1SandboxvolumesIdSnapshotsWithResponse(ctx, apispec.SandboxVolumeID(volumeID), request)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 != nil && resp.JSON201.Data != nil {
		return resp.JSON201.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// ListVolumeSnapshots lists snapshots for a volume.
func (c *Client) ListVolumeSnapshots(ctx context.Context, volumeID string) ([]apispec.Snapshot, error) {
	resp, err := c.api.GetApiV1SandboxvolumesIdSnapshotsWithResponse(ctx, apispec.SandboxVolumeID(volumeID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return *resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// GetVolumeSnapshot gets a snapshot.
func (c *Client) GetVolumeSnapshot(ctx context.Context, volumeID, snapshotID string) (*apispec.Snapshot, error) {
	resp, err := c.api.GetApiV1SandboxvolumesIdSnapshotsSnapshotIdWithResponse(ctx, apispec.SandboxVolumeID(volumeID), apispec.SnapshotID(snapshotID))
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

// DeleteVolumeSnapshot deletes a snapshot.
func (c *Client) DeleteVolumeSnapshot(ctx context.Context, volumeID, snapshotID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := c.api.DeleteApiV1SandboxvolumesIdSnapshotsSnapshotIdWithResponse(ctx, apispec.SandboxVolumeID(volumeID), apispec.SnapshotID(snapshotID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// RestoreVolumeSnapshot restores a snapshot.
func (c *Client) RestoreVolumeSnapshot(ctx context.Context, volumeID, snapshotID string) (*apispec.SuccessRestoreResponse, error) {
	resp, err := c.api.PostApiV1SandboxvolumesIdSnapshotsSnapshotIdRestoreWithResponse(ctx, apispec.SandboxVolumeID(volumeID), apispec.SnapshotID(snapshotID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}
