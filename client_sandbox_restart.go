package sandbox0

import (
	"context"
	"errors"
	"math"
	"regexp"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

var restartKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,255}$`)
var restartIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// RestartSandbox explicitly ends old processes and preserves the published
// writable RootFS. Keep the same generation and key after an uncertain reply.
// Acceptance is asynchronous; inspect the returned operation until terminal.
// This never discovers a fresh generation or falls back from memory restore.
func (c *Client) RestartSandbox(ctx context.Context, sandboxID string, expectedRuntimeGeneration int64, idempotencyKey string) (*apispec.SandboxRestartOperation, error) {
	if expectedRuntimeGeneration < 0 || expectedRuntimeGeneration == math.MaxInt64 || !restartKeyPattern.MatchString(idempotencyKey) {
		return nil, errors.New("invalid restart generation or idempotency key")
	}
	response, err := c.api.RestartSandbox(ctx, &apispec.SandboxRestartRequest{ExpectedRuntimeGeneration: expectedRuntimeGeneration}, apispec.RestartSandboxParams{ID: sandboxID, IdempotencyKey: idempotencyKey})
	if err != nil {
		return nil, err
	}
	r, err := restartOperation(response, sandboxID)
	if err != nil {
		return nil, err
	}
	if r.RequestKey != idempotencyKey || r.FromRuntimeGeneration != expectedRuntimeGeneration {
		return nil, errors.New("restart receipt does not match request")
	}
	return r, nil
}

// GetSandboxRestartOperation reads a historical receipt without guest execution.
func (c *Client) GetSandboxRestartOperation(ctx context.Context, sandboxID, operationID string) (*apispec.SandboxRestartOperation, error) {
	if !restartIDPattern.MatchString(operationID) {
		return nil, errors.New("invalid restart operation ID")
	}
	response, err := c.api.GetSandboxRestartOperation(ctx, apispec.GetSandboxRestartOperationParams{ID: sandboxID, OperationID: operationID})
	if err != nil {
		return nil, err
	}
	r, err := restartOperation(response, sandboxID)
	if err != nil {
		return nil, err
	}
	if r.ID != operationID {
		return nil, errors.New("restart receipt does not match operation")
	}
	return r, nil
}

// FindSandboxRestartOperation recovers a lost submission acknowledgement. A
// missing receipt does not rule out a concurrent submission committing later.
func (c *Client) FindSandboxRestartOperation(ctx context.Context, sandboxID, requestKey string) (*apispec.SandboxRestartOperation, error) {
	if !restartKeyPattern.MatchString(requestKey) {
		return nil, errors.New("invalid restart request key")
	}
	response, err := c.api.FindSandboxRestartOperation(ctx, apispec.FindSandboxRestartOperationParams{ID: sandboxID, RequestKey: requestKey})
	if err != nil {
		return nil, err
	}
	r, err := restartOperation(response, sandboxID)
	if err != nil {
		return nil, err
	}
	if r.RequestKey != requestKey {
		return nil, errors.New("restart receipt does not match request key")
	}
	return r, nil
}

func restartOperation(response any, sandboxID string) (*apispec.SandboxRestartOperation, error) {
	var r apispec.SandboxRestartOperation
	switch v := response.(type) {
	case *apispec.RestartSandboxAccepted:
		r = (*apispec.SuccessSandboxRestartOperationResponse)(v).Data
	case *apispec.RestartSandboxOK:
		r = (*apispec.SuccessSandboxRestartOperationResponse)(v).Data
	case *apispec.SuccessSandboxRestartOperationResponse:
		r = v.Data
	default:
		return nil, apiErrorFromResponse(response)
	}
	if r.SandboxID != sandboxID || !restartIDPattern.MatchString(r.ID) || !restartKeyPattern.MatchString(r.RequestKey) || r.FromRuntimeGeneration < 0 || r.FromRuntimeGeneration == math.MaxInt64 {
		return nil, errors.New("invalid restart receipt")
	}
	generation, set := r.ResultRuntimeGeneration.Get()
	if string(r.State) == "succeeded" {
		if !set || generation != r.FromRuntimeGeneration+1 {
			return nil, errors.New("invalid restart result generation")
		}
	} else if set {
		return nil, errors.New("uncommitted restart result generation")
	}
	return &r, nil
}
