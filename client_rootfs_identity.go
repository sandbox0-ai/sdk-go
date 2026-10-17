package sandbox0

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// GetSandboxRootFSIdentity reads external runtime/storage provenance without
// guest execution or auto-resume. This observation is not credential safety or
// current storage liveness evidence; condition later mutations independently.
func (c *Client) GetSandboxRootFSIdentity(ctx context.Context, sandboxID string) (*apispec.SandboxRootFSIdentity, error) {
	if sandboxID == "" || strings.TrimSpace(sandboxID) != sandboxID {
		return nil, errors.New("invalid sandbox identity")
	}
	response, err := c.api.GetSandboxRootFSIdentity(ctx, apispec.GetSandboxRootFSIdentityParams{ID: sandboxID})
	if err != nil {
		return nil, err
	}
	result, ok := response.(*apispec.SuccessSandboxRootFSIdentityResponse)
	if !ok {
		return nil, apiErrorFromResponse(response)
	}
	identity := result.Data
	if identity.SandboxID != sandboxID || identity.RuntimeGeneration < 0 || identity.LifecycleEpoch < 0 || identity.WriterEpoch < 0 || identity.ManagedFilesSizeBytes < 0 ||
		identity.FilesystemID == "" || identity.HeadGenerationID == "" || identity.BaseArtifactDigest == "" || identity.SourceOciDigest == "" ||
		identity.ObservedAt.IsZero() || identity.SandboxCreatedAt.IsZero() || identity.FilesystemCreatedAt.IsZero() {
		return nil, errors.New("invalid external RootFS identity")
	}
	for _, field := range []apispec.OptString{identity.RuntimeCompatibilityDigest, identity.RetainedRuntimeCompatibilityDigest} {
		if value, ok := field.Get(); ok &&
			(!rootFSRuntimeCompatibilityPattern.MatchString(value) || string(identity.DesiredState) != "active" || identity.ActiveLifecycle || identity.RuntimeGeneration <= 0) {
			return nil, errors.New("invalid external runtime compatibility")
		}
	}
	return &identity, nil
}

var rootFSRuntimeCompatibilityPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
