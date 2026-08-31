//go:build e2e

package sandbox0_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxNetworkPolicy(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	policy, err := sandbox.GetNetworkPolicy(ctx)
	if err != nil {
		t.Fatalf("get network policy failed: %v", err)
	}
	if policy == nil {
		t.Fatalf("network policy was nil")
	}
	if err := updateNetworkPolicyEventually(ctx, sandbox, *policy); err != nil {
		t.Fatalf("update network policy failed: %v", err)
	}
}

func updateNetworkPolicyEventually(
	ctx context.Context,
	sandbox *sandbox0.Sandbox,
	policy apispec.SandboxNetworkPolicy,
) error {
	for {
		_, err := sandbox.UpdateNetworkPolicy(ctx, policy)
		if err == nil {
			return nil
		}
		var apiErr *sandbox0.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 503 {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}
