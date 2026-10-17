package sandbox0

import (
	"context"
	"errors"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// MemoryOperationLookupOptions preserves the original mutation coordinates.
// ExpectedOperationID optionally binds the lookup to an earlier receipt.
type MemoryOperationLookupOptions struct {
	Action                    apispec.LookupSandboxMemoryOperationAction
	ExpectedRuntimeGeneration int64
	IdempotencyKey            string
	CheckpointID              string
	ExpectedOperationID       string
}

// LookupSandboxMemoryOperation reads the original request without dispatching it.
// Current is a database snapshot, not a lease or a source-health attestation.
// Missing evidence and transport errors remain errors; no retry is implicit.
func (c *Client) LookupSandboxMemoryOperation(ctx context.Context, sandboxID string, input MemoryOperationLookupOptions) (*apispec.SandboxMemoryOperation, error) {
	resume := input.Action == "resume"
	if sandboxID == "" || strings.TrimSpace(sandboxID) != sandboxID ||
		(input.Action != "pause" && !resume) || input.ExpectedRuntimeGeneration < 0 || input.ExpectedRuntimeGeneration > 9007199254740991 ||
		(!resume && input.ExpectedRuntimeGeneration == 0) || (resume && input.ExpectedRuntimeGeneration == 9007199254740991) ||
		!restartKeyPattern.MatchString(input.IdempotencyKey) ||
		(!resume && input.CheckpointID != "") || (resume && !conditionalResumeCheckpointPattern.MatchString(input.CheckpointID)) ||
		(input.ExpectedOperationID != "" && !conditionalLifecycleOperationPattern.MatchString(input.ExpectedOperationID)) {
		return nil, errors.New("invalid memory operation lookup request")
	}
	params := apispec.LookupSandboxMemoryOperationParams{ID: sandboxID, Action: input.Action,
		ExpectedRuntimeGeneration: input.ExpectedRuntimeGeneration, IdempotencyKey: input.IdempotencyKey}
	if resume {
		params.CheckpointID = apispec.NewOptString(input.CheckpointID)
	}
	response, err := c.api.LookupSandboxMemoryOperation(ctx, params)
	if err != nil {
		return nil, err
	}
	success, ok := response.(*apispec.SuccessSandboxMemoryOperationResponse)
	if !ok {
		return nil, apiErrorFromResponse(response)
	}
	result := success.Data
	invalid := errors.New("memory operation observation does not match request or terminal evidence")
	checkpoint := result.OperationID
	if resume {
		checkpoint = input.CheckpointID
	}
	if result.SandboxID != sandboxID || string(result.Action) != string(input.Action) || result.FromRuntimeGeneration != input.ExpectedRuntimeGeneration ||
		!conditionalLifecycleOperationPattern.MatchString(result.OperationID) || (input.ExpectedOperationID != "" && result.OperationID != input.ExpectedOperationID) ||
		result.CheckpointID != checkpoint || result.LifecycleEpoch <= 0 || result.LifecycleEpoch > 9007199254740991 {
		return nil, invalid
	}
	generation, hasGeneration := result.ResultRuntimeGeneration.Get()
	consumesGeneration := resume && (result.State == "completed" || result.State == "failed")
	if hasGeneration != consumesGeneration || (hasGeneration && generation != input.ExpectedRuntimeGeneration+1) {
		return nil, invalid
	}
	reason, hasReason := result.Reason.Get()
	switch result.State {
	case "pending":
		if result.Current || hasReason {
			return nil, invalid
		}
	case "completed":
		if hasReason {
			return nil, invalid
		}
	case "canceled":
		expected := "before_capture"
		if resume {
			expected = "before_restore"
		}
		if !hasReason || string(reason) != expected {
			return nil, invalid
		}
	case "failed":
		expected := "capture_failed"
		if resume {
			expected = "restore_failed"
		}
		if !hasReason || string(reason) != expected || (!resume && result.Current) {
			return nil, invalid
		}
	default:
		return nil, invalid
	}
	return &result, nil
}
