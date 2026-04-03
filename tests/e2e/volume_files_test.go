//go:build e2e

package sandbox0_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestVolumeFileOperations(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
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

	baseDir := fmt.Sprintf("/sdk-go-volume-%d", time.Now().UnixNano())
	filePath := baseDir + "/hello.txt"
	movedPath := baseDir + "/moved.txt"

	if _, err := client.MkdirVolumeFile(ctx, volume.ID, baseDir, true); err != nil {
		t.Fatalf("mkdir volume path failed: %v", err)
	}
	if _, err := client.WriteVolumeFile(ctx, volume.ID, filePath, []byte("hello volume")); err != nil {
		t.Fatalf("write volume file failed: %v", err)
	}
	if _, err := client.StatVolumeFile(ctx, volume.ID, filePath); err != nil {
		t.Fatalf("stat volume file failed: %v", err)
	}
	content, err := client.ReadVolumeFile(ctx, volume.ID, filePath)
	if err != nil {
		t.Fatalf("read volume file failed: %v", err)
	}
	if string(content) != "hello volume" {
		t.Fatalf("read volume file = %q, want %q", string(content), "hello volume")
	}
	entries, err := client.ListVolumeFiles(ctx, volume.ID, baseDir)
	if err != nil {
		t.Fatalf("list volume files failed: %v", err)
	}
	foundHello := false
	for _, entry := range entries {
		if name, ok := entry.Name.Get(); ok && name == "hello.txt" {
			foundHello = true
			break
		}
	}
	if !foundHello {
		t.Fatalf("list volume files missing hello.txt: %+v", entries)
	}
	if _, err := client.MoveVolumeFile(ctx, volume.ID, filePath, movedPath); err != nil {
		t.Fatalf("move volume file failed: %v", err)
	}

	conn, _, err := client.ConnectWatchVolumeFile(ctx, volume.ID)
	if err != nil {
		t.Fatalf("connect watch volume file failed: %v", err)
	}
	_ = conn.Close()

	watchCtx, watchCancel := context.WithTimeout(ctx, 30*time.Second)
	defer watchCancel()
	events, errs, unsubscribe, err := client.WatchVolumeFiles(watchCtx, volume.ID, baseDir, true)
	if err != nil {
		t.Fatalf("watch volume files failed: %v", err)
	}
	defer func() {
		_ = unsubscribe()
	}()

	if _, err := client.WriteVolumeFile(ctx, volume.ID, baseDir+"/watch.txt", []byte("watch")); err != nil {
		t.Fatalf("write watch volume file failed: %v", err)
	}

	received := false
	timeout := time.After(10 * time.Second)
	for !received {
		select {
		case event, ok := <-events:
			if !ok {
				received = true
				break
			}
			if event.Path != "" {
				received = true
			}
		case err := <-errs:
			if err != nil {
				t.Fatalf("watch error: %v", err)
			}
		case <-timeout:
			t.Fatalf("timed out waiting for volume watch event")
		}
	}

	if _, err := client.DeleteVolumeFile(ctx, volume.ID, movedPath); err != nil {
		t.Fatalf("delete moved volume file failed: %v", err)
	}
	if _, err := client.DeleteVolumeFile(ctx, volume.ID, baseDir); err != nil {
		t.Fatalf("delete volume dir failed: %v", err)
	}

	if _, err := client.DeleteVolume(ctx, volume.ID); err != nil {
		t.Fatalf("delete volume failed: %v", err)
	}
	deleted = true
}
