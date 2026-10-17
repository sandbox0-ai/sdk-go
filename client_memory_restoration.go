package sandbox0

import (
	"context"
	"errors"
	"regexp"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

var memoryRestorationSandboxPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var memoryRestorationDigestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// GetSandboxMemoryRestoration reads committed same-sandbox memory lineage.
// Bind its capture generation to owned publication/session history before adoption.
// Errors never trigger resume, cold restart, credential publication or retry.
func (c *Client) GetSandboxMemoryRestoration(ctx context.Context, sandboxID string, generation int64) (*apispec.SandboxMemoryRestoration, error) {
	if !memoryRestorationSandboxPattern.MatchString(sandboxID) || generation < 1 {
		return nil, errors.New("invalid memory restoration target")
	}
	response, err := c.api.GetSandboxMemoryRestoration(ctx, apispec.GetSandboxMemoryRestorationParams{ID: sandboxID, RuntimeGeneration: generation})
	if err != nil {
		return nil, err
	}
	result, ok := response.(*apispec.SuccessSandboxMemoryRestorationResponse)
	if !ok {
		return nil, apiErrorFromResponse(response)
	}
	value := result.Data
	invalid := errors.New("invalid external memory restoration")
	if value.SandboxID != sandboxID || value.RuntimeGeneration != generation || value.FilesystemID == "" || !rootFSRuntimeCompatibilityPattern.MatchString(value.RuntimeCompatibilityDigest) || value.ObservedAt.IsZero() || len(value.Steps) < 1 || len(value.Steps) > 64 {
		return nil, invalid
	}
	var previous int64
	for i, step := range value.Steps {
		if step.OperationID == "" || len(step.OperationID) > 512 || step.CaptureOperationID == "" || len(step.CaptureOperationID) > 512 || !memoryRestorationDigestPattern.MatchString(step.RestoreID) || step.FromRuntimeGeneration < 1 || step.ToRuntimeGeneration <= step.FromRuntimeGeneration || step.ManagedFilesSizeBytes < 0 || step.ManagedFilesSizeBytes > 67108864 || (i > 0 && step.FromRuntimeGeneration != previous) {
			return nil, invalid
		}
		previous = step.ToRuntimeGeneration
	}
	if previous != generation {
		return nil, invalid
	}
	return &value, nil
}
