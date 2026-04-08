//go:build e2e

package sandbox0_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxRunAndCmd(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a custom REPL context with specific settings
	customCtx, err := sandbox.CreateContext(ctx, apispec.CreateContextRequest{
		Type: apispec.NewOptProcessType(apispec.ProcessTypeRepl),
		Repl: apispec.NewOptCreateREPLContextRequest(apispec.CreateREPLContextRequest{
			Alias: apispec.NewOptString("python"),
		}),
		Cwd:            apispec.NewOptString("/tmp"),
		EnvVars:        apispec.NewOptCreateContextRequestEnvVars(map[string]string{"SDK_E2E": "true"}),
		TTLSec:         apispec.NewOptInt32(120),
		IdleTimeoutSec: apispec.NewOptInt32(60),
		PtySize:        apispec.NewOptPTYSize(apispec.PTYSize{Rows: apispec.NewOptInt32(24), Cols: apispec.NewOptInt32(80)}),
	})
	if err != nil {
		t.Fatalf("create context failed: %v", err)
	}
	if customCtx.ID == "" {
		t.Fatalf("create context returned empty ID")
	}

	// Run using the custom context
	runResult, err := sandbox.Run(
		ctx,
		"python",
		"print('hello')\n",
		sandbox0.WithContextID(customCtx.ID),
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test REPL stream via WebSocket
	t.Run("repl_stream", func(t *testing.T) {
		ctxResp, err := sandbox.CreateContext(ctx, apispec.CreateContextRequest{
			Type: apispec.NewOptProcessType(apispec.ProcessTypeRepl),
			Repl: apispec.NewOptCreateREPLContextRequest(apispec.CreateREPLContextRequest{
				Alias: apispec.NewOptString("python"),
			}),
		})
		if err != nil {
			t.Fatalf("create repl context failed: %v", err)
		}

		conn, _, err := sandbox.ConnectWSContext(ctx, ctxResp.ID)
		if err != nil {
			t.Fatalf("connect websocket failed: %v", err)
		}
		defer conn.Close()

		// Send input
		msg := map[string]any{
			"type":       "input",
			"data":       "print('stream')\n",
			"request_id": "req-1",
		}
		if err := conn.WriteJSON(msg); err != nil {
			t.Fatalf("write message failed: %v", err)
		}

		// Read output with timeout
		received, err := readWSOutput(ctx, conn, 10*time.Second)
		if err != nil {
			t.Fatalf("read stream error: %v", err)
		}
		if !received {
			t.Fatalf("repl stream did not produce output")
		}
	})

	// Test CMD stream via WebSocket
	t.Run("cmd_stream", func(t *testing.T) {
		ctxResp, err := sandbox.CreateContext(ctx, apispec.CreateContextRequest{
			Type:          apispec.NewOptProcessType(apispec.ProcessTypeCmd),
			Cmd:           apispec.NewOptCreateCMDContextRequest(apispec.CreateCMDContextRequest{Command: []string{"sh", "-c", "echo stream"}}),
			WaitUntilDone: apispec.NewOptBool(false),
		})
		if err != nil {
			t.Fatalf("create cmd context failed: %v", err)
		}
		defer sandbox.DeleteContext(ctx, ctxResp.ID)

		conn, _, err := sandbox.ConnectWSContext(ctx, ctxResp.ID)
		if err != nil {
			t.Fatalf("connect websocket failed: %v", err)
		}
		defer conn.Close()

		received, err := readWSOutput(ctx, conn, 10*time.Second)
		if err != nil {
			t.Fatalf("read stream error: %v", err)
		}
		if !received {
			t.Fatalf("cmd stream did not produce output")
		}
	})

	t.Run("cmd_stream_helper", func(t *testing.T) {
		stream, err := sandbox.CmdStream(
			ctx,
			"echo helper",
			sandbox0.WithCommand([]string{"sh", "-c", "echo helper"}),
		)
		if err != nil {
			t.Fatalf("cmd stream helper failed: %v", err)
		}
		defer stream.Close()

		output, err := stream.Recv()
		if err != nil {
			t.Fatalf("stream recv failed: %v", err)
		}
		if output.Source == "" {
			t.Fatal("stream output source is empty")
		}
		if output.Data == "" {
			t.Fatal("stream output data is empty")
		}

		done, err := stream.Wait()
		if err != nil {
			t.Fatalf("stream wait failed: %v", err)
		}
		if done.ExitCode == nil || *done.ExitCode != 0 {
			t.Fatalf("stream done exit code = %#v, want 0", done.ExitCode)
		}

		if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
			t.Fatalf("stream recv after done = %v, want io.EOF", err)
		}
	})
}

// readWSOutput reads WebSocket messages until context is done, connection closes, or timeout.
// For REPL processes that don't auto-close, it returns success after receiving valid output.
func readWSOutput(ctx context.Context, conn *websocket.Conn, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	conn.SetReadDeadline(deadline)

	received := false
	for {
		select {
		case <-ctx.Done():
			return received, ctx.Err()
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				if isWsClosed(err) {
					return received, nil
				}
				// For timeout errors, return success if we received output
				// (REPL processes don't auto-close the connection)
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					return received, nil
				}
				return received, err
			}
			var msg struct {
				Source string `json:"source"`
				Data   string `json:"data"`
			}
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}
			if msg.Data != "" || msg.Source != "" {
				received = true
			}
		}
	}
}

func isWsClosed(err error) bool {
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway)
}
