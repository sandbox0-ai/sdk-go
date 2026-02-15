//go:build e2e

package sandbox0_test

import (
	"context"
	"testing"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxLifecycle(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sandbox, err := client.ClaimSandbox(
		ctx,
		cfg.template,
		sandbox0.WithSandboxConfig(apispec.SandboxConfig{}),
		sandbox0.WithSandboxTTL(300),
		sandbox0.WithSandboxHardTTL(600),
		sandbox0.WithSandboxWebhook("https://example.com/webhook", "secret"),
		sandbox0.WithSandboxWebhookWatchDir("/workspace"),
		sandbox0.WithSandboxAutoResume(true),
		sandbox0.WithSandboxNetworkPolicy(apispec.TplSandboxNetworkPolicy{
			Mode: apispec.TplSandboxNetworkPolicyModeAllowAll,
		}),
	)
	if err != nil {
		t.Fatalf("claim sandbox failed: %v", err)
	}
	if sandbox == nil || sandbox.ID == "" {
		t.Fatalf("claim sandbox returned empty sandbox")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteSandbox(cleanupCtx, sandbox.ID)
	})

	if _, err := client.GetSandbox(ctx, sandbox.ID); err != nil {
		t.Fatalf("get sandbox failed: %v", err)
	}
	if _, err := client.StatusSandbox(ctx, sandbox.ID); err != nil {
		t.Fatalf("status sandbox failed: %v", err)
	}

	updateRequest := apispec.SandboxUpdateRequest{
		Config: apispec.NewOptSandboxConfig(apispec.SandboxConfig{
			TTL:        apispec.NewOptInt32(300),
			HardTTL:    apispec.NewOptInt32(600),
			AutoResume: apispec.NewOptBool(false),
		}),
	}
	updated, err := client.UpdateSandbox(ctx, sandbox.ID, updateRequest)
	if err != nil {
		t.Fatalf("update sandbox failed: %v", err)
	}
	if updated.GetAutoResume() {
		t.Fatalf("expected sandbox auto_resume to be false after update")
	}

	if _, err := client.PauseSandbox(ctx, sandbox.ID); err != nil {
		t.Fatalf("pause sandbox failed: %v", err)
	}
	if _, err := client.ResumeSandbox(ctx, sandbox.ID); err != nil {
		t.Fatalf("resume sandbox failed: %v", err)
	}
	if _, err := client.RefreshSandbox(ctx, sandbox.ID, nil); err != nil {
		t.Fatalf("refresh sandbox failed: %v", err)
	}
}
