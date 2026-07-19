package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// DeleteVolumeOptions controls delete volume behavior.
type DeleteVolumeOptions struct {
	Force bool
}

// CreateVolume creates a sandbox volume.
func (c *Client) CreateVolume(ctx context.Context, request apispec.CreateSandboxVolumeRequest) (*apispec.SandboxVolume, error) {
	resp, err := c.api.APIV1SandboxvolumesPost(ctx, &request)
	if err != nil {
		return nil, err
	}

	switch response := resp.(type) {
	case *apispec.SuccessSandboxVolumeResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ListVolume lists sandbox volumes.
func (c *Client) ListVolume(ctx context.Context) ([]apispec.SandboxVolume, error) {
	resp, err := c.api.APIV1SandboxvolumesGet(ctx)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxVolumeListResponse:
		return response.Data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// GetVolume retrieves a sandbox volume.
func (c *Client) GetVolume(ctx context.Context, volumeID string) (*apispec.SandboxVolume, error) {
	resp, err := c.api.APIV1SandboxvolumesIDGet(ctx, apispec.APIV1SandboxvolumesIDGetParams{ID: volumeID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxVolumeResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteVolume deletes a sandbox volume.
func (c *Client) DeleteVolume(ctx context.Context, volumeID string) (*apispec.SuccessDeletedResponse, error) {
	return c.DeleteVolumeWithOptions(ctx, volumeID, nil)
}

// DeleteVolumeWithOptions deletes a sandbox volume with optional behavior.
func (c *Client) DeleteVolumeWithOptions(ctx context.Context, volumeID string, options *DeleteVolumeOptions) (*apispec.SuccessDeletedResponse, error) {
	params := apispec.APIV1SandboxvolumesIDDeleteParams{ID: volumeID}
	if options != nil {
		params.Force = apispec.NewOptBool(options.Force)
	}

	resp, err := c.api.APIV1SandboxvolumesIDDelete(ctx, params)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessDeletedResponse:
		return response, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ForkVolume forks a sandbox volume.
func (c *Client) ForkVolume(ctx context.Context, sourceVolumeID string, request *apispec.ForkVolumeRequest) (*apispec.SandboxVolume, error) {
	var req apispec.OptForkVolumeRequest
	if request != nil {
		req = apispec.NewOptForkVolumeRequest(*request)
	}

	resp, err := c.api.APIV1SandboxvolumesIDForkPost(ctx, req, apispec.APIV1SandboxvolumesIDForkPostParams{ID: sourceVolumeID})
	if err != nil {
		return nil, err
	}

	switch response := resp.(type) {
	case *apispec.SuccessSandboxVolumeResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// CreateVolumeSnapshot creates a snapshot for a volume.
func (c *Client) CreateVolumeSnapshot(ctx context.Context, volumeID string, request apispec.CreateSnapshotRequest) (*apispec.Snapshot, error) {
	resp, err := c.api.APIV1SandboxvolumesIDSnapshotsPost(ctx, &request, apispec.APIV1SandboxvolumesIDSnapshotsPostParams{ID: volumeID})
	if err != nil {
		return nil, err
	}

	switch response := resp.(type) {
	case *apispec.SuccessSnapshotResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ListVolumeSnapshots lists snapshots for a volume.
func (c *Client) ListVolumeSnapshots(ctx context.Context, volumeID string) ([]apispec.Snapshot, error) {
	resp, err := c.api.APIV1SandboxvolumesIDSnapshotsGet(ctx, apispec.APIV1SandboxvolumesIDSnapshotsGetParams{ID: volumeID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSnapshotListResponse:
		return response.Data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// GetVolumeSnapshot gets a snapshot.
func (c *Client) GetVolumeSnapshot(ctx context.Context, volumeID, snapshotID string) (*apispec.Snapshot, error) {
	resp, err := c.api.APIV1SandboxvolumesIDSnapshotsSnapshotIDGet(ctx, apispec.APIV1SandboxvolumesIDSnapshotsSnapshotIDGetParams{
		ID:         volumeID,
		SnapshotID: snapshotID,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSnapshotResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteVolumeSnapshot deletes a snapshot.
func (c *Client) DeleteVolumeSnapshot(ctx context.Context, volumeID, snapshotID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := c.api.APIV1SandboxvolumesIDSnapshotsSnapshotIDDelete(ctx, apispec.APIV1SandboxvolumesIDSnapshotsSnapshotIDDeleteParams{
		ID:         volumeID,
		SnapshotID: snapshotID,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessDeletedResponse:
		return response, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// RestoreVolumeSnapshot restores a snapshot.
func (c *Client) RestoreVolumeSnapshot(ctx context.Context, volumeID, snapshotID string) (*apispec.SuccessRestoreResponse, error) {
	resp, err := c.api.APIV1SandboxvolumesIDSnapshotsSnapshotIDRestorePost(ctx, apispec.APIV1SandboxvolumesIDSnapshotsSnapshotIDRestorePostParams{
		ID:         volumeID,
		SnapshotID: snapshotID,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessRestoreResponse:
		return response, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}
