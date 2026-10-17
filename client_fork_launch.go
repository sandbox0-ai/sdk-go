package sandbox0

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// RootFSForkLaunchOptions must retain the original never-run child coordinates
// and key across uncertainty. ExpectedOperationID binds a previously received receipt.
type RootFSForkLaunchOptions struct {
	ForkOperationID, FilesystemID, HeadGenerationID string
	WriterEpoch                                     int64
	IdempotencyKey, ExpectedOperationID             string
}

var forkLaunchCoordinatePattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,512}$`)
var forkLaunchDigestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func validateForkLaunchInput(sandboxID string, q RootFSForkLaunchOptions) error {
	if sandboxID == "" || len(sandboxID) > 512 || strings.TrimSpace(sandboxID) != sandboxID ||
		!forkLaunchCoordinatePattern.MatchString(q.ForkOperationID) || !forkLaunchCoordinatePattern.MatchString(q.FilesystemID) ||
		!forkLaunchCoordinatePattern.MatchString(q.HeadGenerationID) || q.WriterEpoch < 1 || q.WriterEpoch > 9007199254740991 ||
		!restartKeyPattern.MatchString(q.IdempotencyKey) || (q.ExpectedOperationID != "" && !forkLaunchCoordinatePattern.MatchString(q.ExpectedOperationID)) {
		return errors.New("invalid original fork launch conditions")
	}
	return nil
}

func validateForkLaunchReceipt(r *apispec.SandboxRootFSForkLaunch, sandboxID string, q RootFSForkLaunchOptions) (*apispec.SandboxRootFSForkLaunch, error) {
	if r == nil || r.SandboxID != sandboxID || r.ForkOperationID != q.ForkOperationID || r.FilesystemID != q.FilesystemID ||
		r.HeadGenerationID != q.HeadGenerationID || r.WriterEpoch != q.WriterEpoch || r.LifecycleEpoch != 1 ||
		!forkLaunchCoordinatePattern.MatchString(r.OperationID) || (q.ExpectedOperationID != "" && r.OperationID != q.ExpectedOperationID) {
		return nil, errors.New("fork launch receipt does not match original conditions")
	}
	valid := false
	switch r.State {
	case "pending", "aborted":
		valid = r.RuntimeGeneration == 0 && !r.LaunchDigest.IsSet() && !r.CompletedAt.IsSet() && (r.State != "aborted" || !r.Current)
	case "completed":
		valid = r.RuntimeGeneration == 1 && r.LaunchDigest.IsSet() && forkLaunchDigestPattern.MatchString(r.LaunchDigest.Value) && r.CompletedAt.IsSet() && !r.CompletedAt.Value.IsZero()
	}
	if !valid {
		return nil, errors.New("fork launch receipt has invalid completion evidence")
	}
	return r, nil
}

// LaunchSandboxRootFSFork submits exactly one conditional first launch. It does
// not retry, wait, refresh the original conditions, or fall back to resume.
// Current is an owner snapshot, not a lease or credential admission.
func (c *Client) LaunchSandboxRootFSFork(ctx context.Context, sandboxID string, q RootFSForkLaunchOptions) (*apispec.SandboxRootFSForkLaunch, error) {
	if err := validateForkLaunchInput(sandboxID, q); err != nil {
		return nil, err
	}
	response, err := c.api.LaunchSandboxRootFSFork(ctx, &apispec.SandboxRootFSForkLaunchRequest{
		ForkOperationID: q.ForkOperationID, FilesystemID: q.FilesystemID, HeadGenerationID: q.HeadGenerationID, WriterEpoch: q.WriterEpoch,
	}, apispec.LaunchSandboxRootFSForkParams{ID: sandboxID, IdempotencyKey: q.IdempotencyKey})
	if err != nil {
		return nil, err
	}
	switch value := response.(type) {
	case *apispec.LaunchSandboxRootFSForkOK:
		return validateForkLaunchReceipt(&value.Data, sandboxID, q)
	case *apispec.LaunchSandboxRootFSForkAccepted:
		if value.Data.State != "pending" {
			return nil, errors.New("accepted fork launch returned terminal evidence")
		}
		return validateForkLaunchReceipt(&value.Data, sandboxID, q)
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// GetSandboxRootFSForkLaunch only reads the original launch, including retained
// completed history with Current=false. Missing/aborted is not physical cancellation.
func (c *Client) GetSandboxRootFSForkLaunch(ctx context.Context, sandboxID string, q RootFSForkLaunchOptions) (*apispec.SandboxRootFSForkLaunch, error) {
	if err := validateForkLaunchInput(sandboxID, q); err != nil {
		return nil, err
	}
	response, err := c.api.GetSandboxRootFSForkLaunch(ctx, apispec.GetSandboxRootFSForkLaunchParams{
		ID: sandboxID, IdempotencyKey: q.IdempotencyKey, ForkOperationID: q.ForkOperationID,
		FilesystemID: q.FilesystemID, HeadGenerationID: q.HeadGenerationID, WriterEpoch: q.WriterEpoch,
	})
	if err != nil {
		return nil, err
	}
	if value, ok := response.(*apispec.SuccessSandboxRootFSForkLaunchResponse); ok {
		return validateForkLaunchReceipt(&value.Data, sandboxID, q)
	}
	return nil, apiErrorFromResponse(response)
}
