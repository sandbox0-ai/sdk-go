package sandbox0

import (
	"context"
	"errors"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// GetSandboxHealth reads independently sourced health observations. Guest checks
// are advisory; this does not resume, restart, or attest credential safety.
func (c *Client) GetSandboxHealth(ctx context.Context, sandboxID string) (*apispec.SandboxHealth, error) {
	if sandboxID == "" || strings.TrimSpace(sandboxID) != sandboxID {
		return nil, errors.New("invalid sandbox identity")
	}
	response, err := c.api.GetSandboxHealth(ctx, apispec.GetSandboxHealthParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	result, ok := response.(*apispec.SuccessSandboxHealthResponse)
	if !ok {
		return nil, apiErrorFromResponse(response)
	}
	health := result.Data
	if health.SandboxID != sandboxID || health.ObservedAt.IsZero() || health.Validate() != nil {
		return nil, errors.New("invalid sandbox health observation")
	}
	for _, check := range []apispec.SandboxHealthObservation{
		health.Runtime.LaunchAuthority, health.Runtime.Liveness, health.Execution, health.Terminal,
		health.Storage.Identity, health.Storage.WriterAuthority, health.Storage.Io,
	} {
		if check.Source == apispec.SandboxHealthObservationSourceNone {
			if check.State != apispec.SandboxHealthObservationStateUnknown || !check.ObservedAt.IsNull() {
				return nil, errors.New("invalid unobserved sandbox health")
			}
		} else if value, present := check.ObservedAt.Get(); !present || value.IsZero() {
			return nil, errors.New("missing sandbox health observation time")
		}
	}
	return &health, nil
}
