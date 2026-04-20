//go:build e2e

package sandbox0_test

import (
	"bytes"
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
		sandbox0.WithSandboxNetworkPolicy(apispec.SandboxNetworkPolicy{
			Mode: apispec.SandboxNetworkPolicyModeAllowAll,
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
		Config: apispec.NewOptSandboxUpdateConfig(apispec.SandboxUpdateConfig{
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

func TestClaimSandboxWithBootstrapMounts(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	volume, err := client.CreateVolume(ctx, apispec.CreateSandboxVolumeRequest{})
	if err != nil {
		t.Fatalf("create volume failed: %v", err)
	}
	if volume == nil || volume.ID == "" {
		t.Fatalf("create volume returned empty volume")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteVolume(cleanupCtx, volume.ID)
	})

	seedContent := []byte("hello bootstrap claim mount")
	if _, err := client.WriteVolumeFile(ctx, volume.ID, "/claim-bootstrap/hello.txt", seedContent); err != nil {
		t.Fatalf("write volume file failed: %v", err)
	}

	sandbox, err := client.ClaimSandbox(
		ctx,
		cfg.template,
		sandbox0.WithSandboxBootstrapMount(volume.ID, "/workspace/bootstrap-data"),
	)
	if err != nil {
		t.Fatalf("claim sandbox with bootstrap mounts failed: %v", err)
	}
	if sandbox == nil || sandbox.ID == "" {
		t.Fatalf("claim sandbox returned empty sandbox")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteSandbox(cleanupCtx, sandbox.ID)
	})

	if len(sandbox.BootstrapMounts) == 0 {
		t.Fatal("bootstrap mounts missing from sandbox handle")
	}
	if sandbox.BootstrapMounts[0].State != apispec.MountStatusStateMounted {
		t.Fatalf("bootstrap mount state = %q, want %q", sandbox.BootstrapMounts[0].State, apispec.MountStatusStateMounted)
	}

	got, err := sandbox.ReadFile(ctx, "/workspace/bootstrap-data/claim-bootstrap/hello.txt")
	if err != nil {
		t.Fatalf("read mounted bootstrap file failed: %v", err)
	}
	if !bytes.Equal(got, seedContent) {
		t.Fatalf("mounted bootstrap file content = %q, want %q", string(got), string(seedContent))
	}
}
