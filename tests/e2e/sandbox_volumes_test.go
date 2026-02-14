//go:build e2e

package sandbox0_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxMounts(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

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

	mountPoint := fmt.Sprintf("/mnt/sdk-e2e-%d", time.Now().UnixNano())
	mountResp, err := sandbox.Mount(ctx, volume.ID, mountPoint, nil)
	if err != nil {
		t.Fatalf("mount failed: %v", err)
	}
	if _, err := sandbox.MountStatus(ctx); err != nil {
		t.Fatalf("mount status failed: %v", err)
	}
	if _, err := sandbox.Unmount(ctx, volume.ID, mountResp.MountSessionID); err != nil {
		t.Fatalf("unmount failed: %v", err)
	}

	if _, err := client.DeleteVolume(ctx, volume.ID); err != nil {
		t.Fatalf("delete volume failed: %v", err)
	}
	deleted = true
}
