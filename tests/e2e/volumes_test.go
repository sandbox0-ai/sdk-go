//go:build e2e

package sandbox0_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestVolumeAndSnapshotLifecycle(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
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
