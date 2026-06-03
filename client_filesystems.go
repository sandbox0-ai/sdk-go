package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// CreateFilesystem creates a sandbox filesystem.
func (c *Client) CreateFilesystem(ctx context.Context, request apispec.CreateSandboxFilesystemRequest) (*apispec.SandboxFilesystem, error) {
	resp, err := c.api.APIV1SandboxfilesystemsPost(ctx, &request)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxFilesystemResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ListFilesystems lists sandbox filesystems.
func (c *Client) ListFilesystems(ctx context.Context) ([]apispec.SandboxFilesystem, error) {
	resp, err := c.api.APIV1SandboxfilesystemsGet(ctx)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	return resp.Data, nil
}

// GetFilesystem retrieves a sandbox filesystem.
func (c *Client) GetFilesystem(ctx context.Context, filesystemID string) (*apispec.SandboxFilesystem, error) {
	resp, err := c.api.APIV1SandboxfilesystemsIDGet(ctx, apispec.APIV1SandboxfilesystemsIDGetParams{ID: filesystemID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxFilesystemResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteFilesystem deletes a sandbox filesystem.
func (c *Client) DeleteFilesystem(ctx context.Context, filesystemID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := c.api.APIV1SandboxfilesystemsIDDelete(ctx, apispec.APIV1SandboxfilesystemsIDDeleteParams{ID: filesystemID})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ForkFilesystem forks a sandbox filesystem.
func (c *Client) ForkFilesystem(ctx context.Context, sourceFilesystemID string, request *apispec.ForkSandboxFilesystemRequest) (*apispec.SandboxFilesystem, error) {
	var req apispec.OptForkSandboxFilesystemRequest
	if request != nil {
		req = apispec.NewOptForkSandboxFilesystemRequest(*request)
	}
	resp, err := c.api.APIV1SandboxfilesystemsIDForkPost(ctx, req, apispec.APIV1SandboxfilesystemsIDForkPostParams{ID: sourceFilesystemID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxFilesystemResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// CreateFilesystemSnapshot creates a snapshot for a sandbox filesystem.
func (c *Client) CreateFilesystemSnapshot(ctx context.Context, filesystemID string, request apispec.CreateSandboxFilesystemSnapshotRequest) (*apispec.SandboxFilesystemSnapshot, error) {
	resp, err := c.api.APIV1SandboxfilesystemsIDSnapshotsPost(ctx, &request, apispec.APIV1SandboxfilesystemsIDSnapshotsPostParams{ID: filesystemID})
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
	return &data, nil
}

// ListFilesystemSnapshots lists snapshots for a sandbox filesystem.
func (c *Client) ListFilesystemSnapshots(ctx context.Context, filesystemID string) ([]apispec.SandboxFilesystemSnapshot, error) {
	resp, err := c.api.APIV1SandboxfilesystemsIDSnapshotsGet(ctx, apispec.APIV1SandboxfilesystemsIDSnapshotsGetParams{ID: filesystemID})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	return resp.Data, nil
}

// GetFilesystemSnapshot gets a sandbox filesystem snapshot.
func (c *Client) GetFilesystemSnapshot(ctx context.Context, filesystemID, snapshotID string) (*apispec.SandboxFilesystemSnapshot, error) {
	resp, err := c.api.APIV1SandboxfilesystemsIDSnapshotsSnapshotIDGet(ctx, apispec.APIV1SandboxfilesystemsIDSnapshotsSnapshotIDGetParams{
		ID:         filesystemID,
		SnapshotID: snapshotID,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxFilesystemSnapshotResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteFilesystemSnapshot deletes a sandbox filesystem snapshot.
func (c *Client) DeleteFilesystemSnapshot(ctx context.Context, filesystemID, snapshotID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := c.api.APIV1SandboxfilesystemsIDSnapshotsSnapshotIDDelete(ctx, apispec.APIV1SandboxfilesystemsIDSnapshotsSnapshotIDDeleteParams{
		ID:         filesystemID,
		SnapshotID: snapshotID,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RestoreFilesystemSnapshot restores a sandbox filesystem snapshot.
func (c *Client) RestoreFilesystemSnapshot(ctx context.Context, filesystemID, snapshotID string) (*apispec.SandboxFilesystem, error) {
	resp, err := c.api.APIV1SandboxfilesystemsIDSnapshotsSnapshotIDRestorePost(ctx, apispec.APIV1SandboxfilesystemsIDSnapshotsSnapshotIDRestorePostParams{
		ID:         filesystemID,
		SnapshotID: snapshotID,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessSandboxFilesystemResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}
