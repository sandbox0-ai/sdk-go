//go:build e2e

package sandbox0_test

import (
	"context"
	"fmt"
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
	defer func() { _ = closeFn() }()
	runInput <- sandbox0.StreamInput{Type: sandbox0.StreamInputTypeInput, Data: "print('stream')\n"}
	close(runInput)
	time.AfterFunc(1500*time.Millisecond, func() {
		_ = closeFn()
	})

	runReceived, err := readStreamUntilClosed(ctx, outputs, errs, 20*time.Second)
	if err != nil {
		t.Fatalf("run stream error: %v", err)
	}
	if !runReceived {
		t.Fatalf("run stream did not produce output")
	}

	cmdOutputs, cmdErrs, cmdClose, err := sandbox.CmdStream(ctx, "sh -c \"echo stream\"", nil)
	if err != nil {
		t.Fatalf("cmd stream failed: %v", err)
	}
	defer func() { _ = cmdClose() }()
	time.AfterFunc(2*time.Second, func() {
		_ = cmdClose()
	})

	cmdReceived, err := readStreamUntilClosed(ctx, cmdOutputs, cmdErrs, 20*time.Second)
	if err != nil {
		t.Fatalf("cmd stream error: %v", err)
	}
	if !cmdReceived {
		t.Fatalf("cmd stream did not produce output")
	}
}

func readStreamUntilClosed(
	ctx context.Context,
	outputs <-chan sandbox0.StreamOutput,
	errs <-chan error,
	timeout time.Duration,
) (bool, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	received := false
	for {
		select {
		case output, ok := <-outputs:
			if !ok {
				return received, nil
			}
			if output.Data != "" || output.Source != "" {
				received = true
			}
		case err, ok := <-errs:
			if ok && err != nil {
				return received, err
			}
		case <-timer.C:
			return received, fmt.Errorf("stream timed out after %s", timeout)
		case <-ctx.Done():
			return received, ctx.Err()
		}
	}
}
