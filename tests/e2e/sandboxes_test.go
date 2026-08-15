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

func TestSandboxRootFSSnapshotRestoreFork(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	source, err := client.ClaimSandbox(ctx, cfg.template)
	if err != nil {
		t.Fatalf("claim sandbox failed: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteSandbox(cleanupCtx, source.ID)
	})

	const markerPath = "/tmp/sdk-go-rootfs-marker.txt"
	if _, err := source.WriteFile(ctx, markerPath, []byte("rootfs-v1\n")); err != nil {
		t.Fatalf("write v1 marker failed: %v", err)
	}
	if _, err := client.PauseSandbox(ctx, source.ID); err != nil {
		t.Fatalf("pause source failed: %v", err)
	}

	snapshot, err := client.CreateSandboxRootFSSnapshot(ctx, source.ID, &apispec.CreateSandboxRootFSSnapshotRequest{
		Name: apispec.NewOptString("sdk-go-e2e-rootfs"),
	})
	if err != nil {
		t.Fatalf("create rootfs snapshot failed: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteSandboxRootFSSnapshot(cleanupCtx, snapshot.ID)
	})

	snapshots, err := client.ListSandboxRootFSSnapshots(ctx, source.ID)
	if err != nil {
		t.Fatalf("list rootfs snapshots failed: %v", err)
	}
	found := false
	for _, item := range snapshots.Snapshots {
		if item.ID == snapshot.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created snapshot %q not found in list", snapshot.ID)
	}
	if fetched, err := client.GetSandboxRootFSSnapshot(ctx, snapshot.ID); err != nil {
		t.Fatalf("get rootfs snapshot failed: %v", err)
	} else if fetched.ID != snapshot.ID {
		t.Fatalf("fetched snapshot ID = %q, want %q", fetched.ID, snapshot.ID)
	}

	if _, err := client.ResumeSandbox(ctx, source.ID); err != nil {
		t.Fatalf("resume source failed: %v", err)
	}
	if _, err := source.WriteFile(ctx, markerPath, []byte("rootfs-v2\n")); err != nil {
		t.Fatalf("write v2 marker failed: %v", err)
	}
	if _, err := client.PauseSandbox(ctx, source.ID); err != nil {
		t.Fatalf("pause source for restore failed: %v", err)
	}
	if restored, err := client.RestoreSandboxRootFS(ctx, source.ID, apispec.RestoreSandboxRootFSRequest{SnapshotID: snapshot.ID}); err != nil {
		t.Fatalf("restore rootfs failed: %v", err)
	} else if restored.SnapshotID != snapshot.ID {
		t.Fatalf("restored snapshot ID = %q, want %q", restored.SnapshotID, snapshot.ID)
	}

	forked, err := client.ForkSandbox(ctx, source.ID, nil)
	if err != nil {
		t.Fatalf("fork sandbox failed: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteSandbox(cleanupCtx, forked.Sandbox.ID)
	})
	if forked.SourceSandboxID != source.ID {
		t.Fatalf("fork source ID = %q, want %q", forked.SourceSandboxID, source.ID)
	}

	if _, err := client.DeleteSandboxRootFSSnapshot(ctx, snapshot.ID); err != nil {
		t.Fatalf("delete rootfs snapshot failed: %v", err)
	}
	if _, err := client.ResumeSandbox(ctx, source.ID); err != nil {
		t.Fatalf("resume restored source failed: %v", err)
	}
	if _, err := client.ResumeSandbox(ctx, forked.Sandbox.ID); err != nil {
		t.Fatalf("resume fork failed: %v", err)
	}

	sourceContent, err := source.ReadFile(ctx, markerPath)
	if err != nil {
		t.Fatalf("read source marker failed: %v", err)
	}
	if !bytes.Equal(sourceContent, []byte("rootfs-v1\n")) {
		t.Fatalf("source marker = %q, want rootfs-v1", string(sourceContent))
	}
	forkContent, err := client.Sandbox(forked.Sandbox.ID).ReadFile(ctx, markerPath)
	if err != nil {
		t.Fatalf("read fork marker failed: %v", err)
	}
	if !bytes.Equal(forkContent, []byte("rootfs-v1\n")) {
		t.Fatalf("fork marker = %q, want rootfs-v1", string(forkContent))
	}
}
