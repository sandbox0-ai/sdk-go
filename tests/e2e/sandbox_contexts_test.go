//go:build e2e

package sandbox0_test

import (
	"context"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxContextOperations(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	createReq := apispec.CreateContextRequest{
		Type: apispec.NewOptProcessType(apispec.ProcessTypeRepl),
		Repl: apispec.NewOptCreateREPLContextRequest(apispec.CreateREPLContextRequest{
			Language: apispec.NewOptString("python"),
		}),
		PtySize: apispec.NewOptPTYSize(apispec.PTYSize{
			Rows: apispec.NewOptInt32(24),
			Cols: apispec.NewOptInt32(80),
		}),
	}
	contextResp, err := sandbox.CreateContext(ctx, createReq)
	if err != nil {
		t.Fatalf("create context failed: %v", err)
	}
	if contextResp == nil || contextResp.ID == "" {
		t.Fatalf("create context returned empty response")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = sandbox.DeleteContext(cleanupCtx, contextResp.ID)
	})

	if _, err := sandbox.ListContext(ctx); err != nil {
		t.Fatalf("list context failed: %v", err)
	}
	if _, err := sandbox.GetContext(ctx, contextResp.ID); err != nil {
		t.Fatalf("get context failed: %v", err)
	}
	if _, err := sandbox.ContextInput(ctx, contextResp.ID, "print('hi')\n"); err != nil {
		t.Fatalf("context input failed: %v", err)
	}
	if _, err := sandbox.ContextExec(ctx, contextResp.ID, "print('exec')\n"); err != nil {
		t.Fatalf("context exec failed: %v", err)
	}
	if _, err := sandbox.ContextResize(ctx, contextResp.ID, 40, 100); err != nil {
		t.Fatalf("context resize failed: %v", err)
	}
	if _, err := sandbox.ContextSignal(ctx, contextResp.ID, "SIGINT"); err != nil {
		t.Fatalf("context signal failed: %v", err)
	}
	if _, err := sandbox.ContextStats(ctx, contextResp.ID); err != nil {
		t.Fatalf("context stats failed: %v", err)
	}
	if _, err := sandbox.RestartContext(ctx, contextResp.ID); err != nil {
		t.Fatalf("restart context failed: %v", err)
	}
	conn, _, err := sandbox.ConnectWSContext(ctx, contextResp.ID)
	if err != nil {
		t.Fatalf("connect ws context failed: %v", err)
	}
	_ = conn.Close()

	cmdReq := apispec.CreateContextRequest{
		Type: apispec.NewOptProcessType(apispec.ProcessTypeCmd),
		Cmd: apispec.NewOptCreateCMDContextRequest(apispec.CreateCMDContextRequest{
			Command: []string{"/bin/sh", "-lc", "echo sdk-e2e"},
		}),
		WaitUntilDone: apispec.NewOptBool(true),
	}
	cmdResp, err := sandbox.CreateContext(ctx, cmdReq)
	if err != nil {
		t.Fatalf("create cmd context failed: %v", err)
	}
	if cmdResp == nil || cmdResp.ID == "" {
		t.Fatalf("create cmd context returned empty response")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = sandbox.DeleteContext(cleanupCtx, cmdResp.ID)
	})
}
