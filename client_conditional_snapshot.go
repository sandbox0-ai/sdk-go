package sandbox0

import (
	"context"
	"errors"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// CaptureSandboxRootFSSnapshot submits one generation-conditioned running
// capture. On uncertainty retain the same key, generation and metadata.
func (c *Client) CaptureSandboxRootFSSnapshot(ctx context.Context, sandboxID string, request *apispec.CaptureSandboxRootFSSnapshotRequest, idempotencyKey string) (*apispec.SandboxRootFSSnapshot, error) {
	if sandboxID == "" || request == nil || request.ExpectedRuntimeGeneration <= 0 || request.ExpectedRuntimeGeneration > 9007199254740991 || !restartKeyPattern.MatchString(idempotencyKey) {
		return nil, errors.New("conditional snapshot requires a source, positive safe generation and canonical key")
	}
	response, err := c.api.CaptureSandboxRootFSSnapshot(ctx, request, apispec.CaptureSandboxRootFSSnapshotParams{ID: sandboxID, IdempotencyKey: idempotencyKey})
	if err != nil {
		return nil, err
	}
	if value, ok := response.(*apispec.SuccessSandboxRootFSSnapshotResponse); ok {
		if value.Data.ID == "" || value.Data.SandboxID != sandboxID || value.Data.CreatedAt.IsZero() {
			return nil, errors.New("conditional snapshot returned mismatched identity")
		}
		return &value.Data, nil
	}
	return nil, apiErrorFromResponse(response)
}

// GetCapturedSandboxRootFSSnapshot only reads the original receipt. All missing,
// pending or unavailable outcomes remain errors; none authorizes a replacement.
func (c *Client) GetCapturedSandboxRootFSSnapshot(ctx context.Context, sandboxID string, expectedRuntimeGeneration int64, idempotencyKey string) (*apispec.SandboxRootFSSnapshot, error) {
	if sandboxID == "" || expectedRuntimeGeneration <= 0 || expectedRuntimeGeneration > 9007199254740991 || !restartKeyPattern.MatchString(idempotencyKey) {
		return nil, errors.New("snapshot observation requires a source, positive safe generation and canonical key")
	}
	response, err := c.api.GetCapturedSandboxRootFSSnapshot(ctx, apispec.GetCapturedSandboxRootFSSnapshotParams{ID: sandboxID, ExpectedRuntimeGeneration: expectedRuntimeGeneration, IdempotencyKey: idempotencyKey})
	if err != nil {
		return nil, err
	}
	if value, ok := response.(*apispec.SuccessSandboxRootFSSnapshotResponse); ok {
		if value.Data.ID == "" || value.Data.SandboxID != sandboxID || value.Data.CreatedAt.IsZero() {
			return nil, errors.New("snapshot observation returned mismatched identity")
		}
		return &value.Data, nil
	}
	return nil, apiErrorFromResponse(response)
}
