package sandbox0

import (
	"context"
	"errors"
	"regexp"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

var conditionalResumeCheckpointPattern = regexp.MustCompile(`^[!-~]{1,256}$`)

// ResumeSandboxMemoryForOperation restores an exact retained checkpoint from
// the expected source generation. Retain all inputs after an uncertain reply.
// This never refreshes the generation, retries automatically, polls generic
// running state or starts filesystem-only replacement processes.
func (c *Client) ResumeSandboxMemoryForOperation(ctx context.Context, sandboxID string, expectedRuntimeGeneration int64, checkpointID, idempotencyKey string) (*apispec.ResumeSandboxResponse, error) {
	if expectedRuntimeGeneration < 0 || expectedRuntimeGeneration >= 9007199254740991 || !conditionalResumeCheckpointPattern.MatchString(checkpointID) || !restartKeyPattern.MatchString(idempotencyKey) {
		return nil, errors.New("invalid conditional memory resume generation, checkpoint or idempotency key")
	}
	response, err := c.api.APIV1SandboxesIDResumePost(ctx, apispec.NewOptSandboxExecutionStateRequest(apispec.SandboxExecutionStateRequest{
		Memory: apispec.NewOptBool(true), ExpectedRuntimeGeneration: apispec.NewOptInt64(expectedRuntimeGeneration), CheckpointID: apispec.NewOptString(checkpointID),
	}), apispec.APIV1SandboxesIDResumePostParams{ID: sandboxID, IdempotencyKey: apispec.NewOptString(idempotencyKey)})
	if err != nil {
		return nil, err
	}
	result, ok := response.(*apispec.SuccessResumeSandboxResponse)
	if !ok {
		return nil, apiErrorFromResponse(response)
	}
	data := result.Data
	operation, operationSet := data.OperationID.Get()
	checkpoint, checkpointSet := data.CheckpointID.Get()
	from, fromSet := data.FromRuntimeGeneration.Get()
	generation, generationSet := data.RuntimeGeneration.Get()
	if data.SandboxID != sandboxID || !data.Resumed || !operationSet || !conditionalLifecycleOperationPattern.MatchString(operation) ||
		!checkpointSet || checkpoint != checkpointID || !fromSet || from != expectedRuntimeGeneration || !generationSet || generation != expectedRuntimeGeneration+1 {
		return nil, errors.New("conditional memory resume response does not match request")
	}
	return &data, nil
}
