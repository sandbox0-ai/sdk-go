package sandbox0

import (
	"context"
	"errors"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// GetSandboxRootFSFork reads the original unchanged, never-run RootFS child.
// Missing/pending outcomes remain errors; it never retries or submits a fork.
// The receipt is not a reservation or a credential-safety attestation.
func (c *Client) GetSandboxRootFSFork(ctx context.Context, sandboxID string, expectedRuntimeGeneration int64, idempotencyKey string) (*apispec.SandboxRootFSForkObservation, error) {
	if sandboxID == "" || expectedRuntimeGeneration < 0 || expectedRuntimeGeneration > 9007199254740991 || !restartKeyPattern.MatchString(idempotencyKey) {
		return nil, errors.New("fork observation requires a source, nonnegative safe generation and canonical key")
	}
	response, err := c.api.GetSandboxRootFSFork(ctx, apispec.GetSandboxRootFSForkParams{ID: sandboxID, ExpectedRuntimeGeneration: expectedRuntimeGeneration, IdempotencyKey: idempotencyKey})
	if err != nil {
		return nil, err
	}
	if value, ok := response.(*apispec.SuccessSandboxRootFSForkObservationResponse); ok {
		r := &value.Data
		if r.SourceSandboxID != sandboxID || r.SourceRuntimeGeneration != expectedRuntimeGeneration || r.TargetSandboxID == sandboxID ||
			r.TargetFilesystemID != r.TargetSandboxID || r.SandboxCreatedAt.IsZero() || r.FilesystemCreatedAt.IsZero() || r.ObservedAt.IsZero() {
			return nil, errors.New("fork observation returned mismatched identity")
		}
		return r, nil
	}
	return nil, apiErrorFromResponse(response)
}
