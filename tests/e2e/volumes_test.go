//go:build e2e

package sandbox0_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestVolumeAndSnapshotLifecycle(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	volume, err := client.CreateVolume(ctx, apispec.CreateSandboxVolumeRequest{})
	if err != nil {
		t.Fatalf("create volume failed: %v", err)
	}
	if volume == nil || volume.ID == "" {
		t.Fatalf("create volume returned empty volume")
	}
	deleted := false
	t.Cleanup(func() {
		if deleted {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteVolume(cleanupCtx, volume.ID)
	})

	if _, err := client.ListVolume(ctx); err != nil {
		t.Fatalf("list volume failed: %v", err)
	}
	if _, err := client.GetVolume(ctx, volume.ID); err != nil {
		t.Fatalf("get volume failed: %v", err)
	}

	snapshotName := fmt.Sprintf("sdk-e2e-snap-%d", time.Now().UnixNano())
	snapshot, err := client.CreateVolumeSnapshot(ctx, volume.ID, apispec.CreateSnapshotRequest{
		Name:        snapshotName,
		Description: apispec.NewOptString("sdk e2e snapshot"),
	})
	if err != nil {
		t.Fatalf("create snapshot failed: %v", err)
	}
	if snapshot == nil || snapshot.ID == "" {
		t.Fatalf("create snapshot returned empty snapshot")
	}

	if _, err := client.ListVolumeSnapshots(ctx, volume.ID); err != nil {
		t.Fatalf("list volume snapshots failed: %v", err)
	}
	if _, err := client.GetVolumeSnapshot(ctx, volume.ID, snapshot.ID); err != nil {
		t.Fatalf("get volume snapshot failed: %v", err)
	}
	if _, err := client.RestoreVolumeSnapshot(ctx, volume.ID, snapshot.ID); err != nil {
		t.Fatalf("restore volume snapshot failed: %v", err)
	}
	if _, err := client.DeleteVolumeSnapshot(ctx, volume.ID, snapshot.ID); err != nil {
		t.Fatalf("delete volume snapshot failed: %v", err)
	}

	if _, err := client.DeleteVolume(ctx, volume.ID); err != nil {
		t.Fatalf("delete volume failed: %v", err)
	}
	deleted = true
}

func TestForkVolumeIsolation(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sourceVolume, err := client.CreateVolume(ctx, apispec.CreateSandboxVolumeRequest{})
	if err != nil {
		t.Fatalf("create source volume failed: %v", err)
	}
	if sourceVolume == nil || sourceVolume.ID == "" {
		t.Fatalf("create source volume returned empty volume")
	}
	sourceDeleted := false
	t.Cleanup(func() {
		if sourceDeleted {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteVolume(cleanupCtx, sourceVolume.ID)
	})

	initialMountPoint := fmt.Sprintf("/mnt/src-init-%d", time.Now().UnixNano())
	initialMount, err := sandbox.Mount(ctx, sourceVolume.ID, initialMountPoint, nil)
	if err != nil {
		t.Fatalf("mount source volume failed: %v", err)
	}
	if _, err := sandbox.WriteFile(ctx, initialMountPoint+"/hello.txt", []byte("source-original\n")); err != nil {
		t.Fatalf("write source file failed: %v", err)
	}
	if _, err := sandbox.Unmount(ctx, sourceVolume.ID, initialMount.MountSessionID); err != nil {
		t.Fatalf("unmount source volume failed: %v", err)
	}

	forkedVolume, err := client.ForkVolume(ctx, sourceVolume.ID, nil)
	if err != nil {
		t.Fatalf("fork volume failed: %v", err)
	}
	if forkedVolume == nil || forkedVolume.ID == "" {
		t.Fatalf("fork volume returned empty volume")
	}
	forkDeleted := false
	t.Cleanup(func() {
		if forkDeleted {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteVolume(cleanupCtx, forkedVolume.ID)
	})

	sourceMountPoint := fmt.Sprintf("/mnt/src-%d", time.Now().UnixNano())
	sourceMount, err := sandbox.Mount(ctx, sourceVolume.ID, sourceMountPoint, nil)
	if err != nil {
		t.Fatalf("mount source volume for verification failed: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = sandbox.Unmount(cleanupCtx, sourceVolume.ID, sourceMount.MountSessionID)
	})

	forkMountPoint := fmt.Sprintf("/mnt/fork-%d", time.Now().UnixNano())
	forkMount, err := sandbox.Mount(ctx, forkedVolume.ID, forkMountPoint, nil)
	if err != nil {
		t.Fatalf("mount forked volume failed: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = sandbox.Unmount(cleanupCtx, forkedVolume.ID, forkMount.MountSessionID)
	})

	if _, err := sandbox.WriteFile(ctx, forkMountPoint+"/hello.txt", []byte("fork-updated\n")); err != nil {
		t.Fatalf("write fork file failed: %v", err)
	}

	sourceContent, err := sandbox.ReadFile(ctx, sourceMountPoint+"/hello.txt")
	if err != nil {
		t.Fatalf("read source file failed: %v", err)
	}
	if string(sourceContent) != "source-original\n" {
		t.Fatalf("source volume was modified by fork write, got %q", string(sourceContent))
	}

	if _, err := sandbox.Unmount(ctx, sourceVolume.ID, sourceMount.MountSessionID); err != nil {
		t.Fatalf("unmount source verification mount failed: %v", err)
	}
	if _, err := sandbox.Unmount(ctx, forkedVolume.ID, forkMount.MountSessionID); err != nil {
		t.Fatalf("unmount fork verification mount failed: %v", err)
	}

	if _, err := client.DeleteVolume(ctx, forkedVolume.ID); err != nil {
		t.Fatalf("delete forked volume failed: %v", err)
	}
	forkDeleted = true

	if _, err := client.DeleteVolume(ctx, sourceVolume.ID); err != nil {
		t.Fatalf("delete source volume failed: %v", err)
	}
	sourceDeleted = true
}
