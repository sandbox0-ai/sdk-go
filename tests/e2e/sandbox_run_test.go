//go:build e2e

package sandbox0_test

import (
	"context"
	"testing"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
)

func TestSandboxRunAndCmd(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	runResult, err := sandbox.Run(
		ctx,
		"python",
		"print('hello')\n",
		sandbox0.WithContextTTL(120),
		sandbox0.WithIdleTimeout(60),
		sandbox0.WithCWD("/tmp"),
		sandbox0.WithEnvVars(map[string]string{"SDK_E2E": "true"}),
		sandbox0.WithPTYSize(24, 80),
	)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if runResult.ContextID == "" {
		t.Fatalf("run returned empty context ID")
	}

	if _, err := sandbox.Run(ctx, "python", "print('reuse')\n", sandbox0.WithContextID(runResult.ContextID)); err != nil {
		t.Fatalf("run with context ID failed: %v", err)
	}

	cmdResult, err := sandbox.Cmd(
		ctx,
		"echo hello",
		sandbox0.WithCommand([]string{"sh", "-c", "echo hello"}),
		sandbox0.WithCmdTTL(120),
		sandbox0.WithCmdIdleTimeout(60),
		sandbox0.WithCmdCWD("/tmp"),
		sandbox0.WithCmdEnvVars(map[string]string{"SDK_E2E_CMD": "true"}),
		sandbox0.WithCmdPTYSize(24, 80),
	)
	if err != nil {
		t.Fatalf("cmd failed: %v", err)
	}
	if cmdResult.ContextID == "" {
		t.Fatalf("cmd returned empty context ID")
	}
}

func TestSandboxStreams(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	runInput := make(chan sandbox0.StreamInput, 1)
	outputs, errs, closeFn, err := sandbox.RunStream(ctx, "python", runInput)
	if err != nil {
		t.Fatalf("run stream failed: %v", err)
	}
	runInput <- sandbox0.StreamInput{Type: sandbox0.StreamInputTypeInput, Data: "print('stream')\n"}

	runReceived := false
	select {
	case <-outputs:
		runReceived = true
	case err := <-errs:
		if err != nil {
			t.Fatalf("run stream error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("run stream timed out")
	}
	if !runReceived {
		t.Fatalf("run stream did not produce output")
	}
	_ = closeFn()

	cmdInput := make(chan sandbox0.StreamInput, 1)
	cmdOutputs, cmdErrs, cmdClose, err := sandbox.CmdStream(ctx, "sh -c \"echo stream\"", cmdInput)
	if err != nil {
		t.Fatalf("cmd stream failed: %v", err)
	}

	cmdReceived := false
	select {
	case <-cmdOutputs:
		cmdReceived = true
	case err := <-cmdErrs:
		if err != nil {
			t.Fatalf("cmd stream error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("cmd stream timed out")
	}
	if !cmdReceived {
		t.Fatalf("cmd stream did not produce output")
	}
	_ = cmdClose()
}
