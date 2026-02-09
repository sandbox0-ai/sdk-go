//go:build e2e

package sandbox0_test

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSandboxFileOperations(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	baseDir := fmt.Sprintf("/tmp/sdk-e2e-%d", time.Now().UnixNano())
	filePath := baseDir + "/hello.txt"
	movedPath := baseDir + "/moved.txt"

	if _, err := sandbox.Mkdir(ctx, baseDir, true); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if _, err := sandbox.WriteFile(ctx, filePath, []byte("hello e2e")); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	if _, err := sandbox.StatFile(ctx, filePath); err != nil {
		t.Fatalf("stat file failed: %v", err)
	}
	if _, err := sandbox.ReadFile(ctx, filePath); err != nil {
		t.Fatalf("read file failed: %v", err)
	}
	if _, err := sandbox.ListFiles(ctx, baseDir); err != nil {
		t.Fatalf("list files failed: %v", err)
	}
	if _, err := sandbox.MoveFile(ctx, filePath, movedPath); err != nil {
		t.Fatalf("move file failed: %v", err)
	}

	conn, _, err := sandbox.ConnectWatchFile(ctx)
	if err != nil {
		t.Fatalf("connect watch file failed: %v", err)
	}
	_ = conn.Close()

	watchCtx, watchCancel := context.WithTimeout(ctx, 30*time.Second)
	defer watchCancel()
	events, errs, unsubscribe, err := sandbox.WatchFiles(watchCtx, baseDir, true)
	if err != nil {
		t.Fatalf("watch files failed: %v", err)
	}
	defer func() {
		_ = unsubscribe()
	}()

	if _, err := sandbox.WriteFile(ctx, baseDir+"/watch.txt", []byte("watch")); err != nil {
		t.Fatalf("write watch file failed: %v", err)
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
			t.Fatalf("timed out waiting for watch event")
		}
	}

	if _, err := sandbox.DeleteFile(ctx, movedPath); err != nil {
		t.Fatalf("delete file failed: %v", err)
	}
	if _, err := sandbox.DeleteFile(ctx, baseDir); err != nil {
		t.Fatalf("delete dir failed: %v", err)
	}
}
