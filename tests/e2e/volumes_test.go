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

	if _, err := client.WriteVolumeFile(ctx, sourceVolume.ID, "/hello.txt", []byte("source-original\n")); err != nil {
		t.Fatalf("write source file failed: %v", err)
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

	if _, err := client.WriteVolumeFile(ctx, forkedVolume.ID, "/hello.txt", []byte("fork-updated\n")); err != nil {
		t.Fatalf("write fork file failed: %v", err)
	}

	sourceContent, err := client.ReadVolumeFile(ctx, sourceVolume.ID, "/hello.txt")
	if err != nil {
		t.Fatalf("read source file failed: %v", err)
	}
	if string(sourceContent) != "source-original\n" {
		t.Fatalf("source volume was modified by fork write, got %q", string(sourceContent))
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
