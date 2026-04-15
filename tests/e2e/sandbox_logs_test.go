//go:build e2e

package sandbox0_test

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
)

func TestSandboxLogsIncludeProcessOutputAndStreamPlainText(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

	snapshotStdoutMarker := fmt.Sprintf("sdk-go-log-snapshot-stdout-%d", time.Now().UnixNano())
	snapshotStderrMarker := snapshotStdoutMarker + "-stderr"
	runShellCommand(t, sandbox, fmt.Sprintf("printf '%s\\n'; printf '%s\\n' >&2", snapshotStdoutMarker, snapshotStderrMarker))

	logs := waitForSandboxLogs(t, sandbox, snapshotStdoutMarker, snapshotStderrMarker)
	if logs.SandboxID != sandbox.ID {
		t.Fatalf("logs sandbox_id = %q, want %q", logs.SandboxID, sandbox.ID)
	}
	if logs.PodName == "" {
		t.Fatal("logs missing pod name")
	}
	if logs.Previous {
		t.Fatal("logs previous = true, want false")
	}

	followMarker := fmt.Sprintf("sdk-go-log-follow-%d", time.Now().UnixNano())
	followCmd := runShellCommandAsync(t, sandbox, fmt.Sprintf("sleep 2; printf '%s\\n'", followMarker))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = sandbox.DeleteContext(ctx, followCmd.ContextID)
	})

	followCtx, cancelFollow := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelFollow()
	stream, err := sandbox.StreamLogs(followCtx, nil)
	if err != nil {
		t.Fatalf("stream sandbox logs failed: %v", err)
	}
	defer stream.Close()
	if stream.SandboxID != sandbox.ID {
		t.Fatalf("stream sandbox_id = %q, want %q", stream.SandboxID, sandbox.ID)
	}
	if stream.PodName == "" {
		t.Fatal("stream missing pod name")
	}

	followLines := make(chan string, 1)
	readErrs := make(chan error, 1)
	go scanUntilContains(stream, followMarker, followLines, readErrs)

	select {
	case line := <-followLines:
		if !strings.Contains(line, followMarker) {
			t.Fatalf("follow line = %q, want marker %q", line, followMarker)
		}
	case err := <-readErrs:
		if err == nil {
			t.Fatalf("log stream closed before marker %q", followMarker)
		}
		t.Fatalf("read log stream failed: %v", err)
	case <-followCtx.Done():
		t.Fatalf("timed out waiting for followed log marker %q", followMarker)
	}
}

func runShellCommand(t *testing.T, sandbox *sandbox0.Sandbox, command string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := sandbox.Cmd(ctx, "sh", sandbox0.WithCommand([]string{"sh", "-c", command})); err != nil {
		t.Fatalf("run sandbox command failed: %v", err)
	}
}

func runShellCommandAsync(t *testing.T, sandbox *sandbox0.Sandbox, command string) sandbox0.CmdResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := sandbox.Cmd(ctx, "sh", sandbox0.WithCommand([]string{"sh", "-c", command}), sandbox0.WithCmdWait(false))
	if err != nil {
		t.Fatalf("run async sandbox command failed: %v", err)
	}
	return result
}

func waitForSandboxLogs(t *testing.T, sandbox *sandbox0.Sandbox, markers ...string) *sandbox0.SandboxLogs {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tailLines := int64(100)

	for {
		logs, err := sandbox.GetLogs(ctx, &sandbox0.SandboxLogsOptions{TailLines: &tailLines})
		if err != nil {
			t.Fatalf("get sandbox logs failed: %v", err)
		}
		if containsAll(logs.Logs, markers...) {
			return logs
		}

		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for log markers %q in logs:\n%s", markers, logs.Logs)
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func containsAll(text string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			return false
		}
	}
	return true
}

func scanUntilContains(stream *sandbox0.SandboxLogsStream, marker string, lines chan<- string, errs chan<- error) {
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, marker) {
			lines <- line
			return
		}
	}
	errs <- scanner.Err()
}
