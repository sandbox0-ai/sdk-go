package sandbox0

import (
	"context"
	"errors"
	"regexp"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

var conditionalLifecycleOperationPattern = regexp.MustCompile(`^idem-[a-f0-9]{64}$`)

// PauseSandboxMemoryForOperation binds memory pause to one source generation.
// Retain the same generation and key after an uncertain reply. Paused=false
// means capture is still pending. This never refreshes the generation, polls
// unrelated lifecycle state, or falls back to filesystem-only pause.
func (c *Client) PauseSandboxMemoryForOperation(ctx context.Context, sandboxID string, expectedRuntimeGeneration int64, idempotencyKey string) (*apispec.PauseSandboxResponse, error) {
	if expectedRuntimeGeneration <= 0 || expectedRuntimeGeneration > 9007199254740991 || !restartKeyPattern.MatchString(idempotencyKey) {
		return nil, errors.New("invalid conditional memory pause generation or idempotency key")
	}
	response, err := c.api.APIV1SandboxesIDPausePost(ctx, apispec.NewOptSandboxPauseRequest(apispec.SandboxPauseRequest{
		Memory: apispec.NewOptBool(true), ExpectedRuntimeGeneration: apispec.NewOptInt64(expectedRuntimeGeneration),
	}), apispec.APIV1SandboxesIDPausePostParams{ID: sandboxID, IdempotencyKey: apispec.NewOptString(idempotencyKey)})
	if err != nil {
		return nil, err
	}
	var result apispec.PauseSandboxResponse
	switch value := response.(type) {
	case *apispec.APIV1SandboxesIDPausePostOK:
		result = (*apispec.SuccessPauseSandboxResponse)(value).Data
	case *apispec.APIV1SandboxesIDPausePostAccepted:
		result = (*apispec.SuccessPauseSandboxResponse)(value).Data
	default:
		return nil, apiErrorFromResponse(response)
	}
	operation, operationSet := result.OperationID.Get()
	generation, generationSet := result.RuntimeGeneration.Get()
	if result.SandboxID != sandboxID || !operationSet || !conditionalLifecycleOperationPattern.MatchString(operation) || !generationSet || generation != expectedRuntimeGeneration {
		return nil, errors.New("conditional memory pause response does not match request")
	}
	return &result, nil
}
