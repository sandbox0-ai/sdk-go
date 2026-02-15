//go:build e2e

package sandbox0_test

import (
	"context"
	"testing"
	"time"
)

func TestSandboxNetworkPolicy(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	policy, err := sandbox.GetNetworkPolicy(ctx)
	if err != nil {
		t.Fatalf("get network policy failed: %v", err)
	}
	if policy == nil {
		t.Fatalf("network policy was nil")
	}
	if _, err := sandbox.UpdateNetworkPolicy(ctx, *policy); err != nil {
		t.Fatalf("update network policy failed: %v", err)
	}
}
