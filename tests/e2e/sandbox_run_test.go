//go:build e2e

package sandbox0_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	// Test REPL stream via WebSocket
	t.Run("repl_stream", func(t *testing.T) {
		ctxResp, err := sandbox.CreateContext(ctx, apispec.CreateContextRequest{
			Type: apispec.NewOptProcessType(apispec.ProcessTypeRepl),
			Repl: apispec.NewOptCreateREPLContextRequest(apispec.CreateREPLContextRequest{
				Language: apispec.NewOptString("python"),
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
		received, err := readWSUntilClosed(ctx, conn, 10*time.Second)
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

		received, err := readWSUntilClosed(ctx, conn, 10*time.Second)
		if err != nil {
			t.Fatalf("read stream error: %v", err)
		}
		if !received {
			t.Fatalf("cmd stream did not produce output")
		}
	})
}

func readWSUntilClosed(ctx context.Context, conn *websocket.Conn, timeout time.Duration) (bool, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	received := false
	for {
		select {
		case <-timer.C:
			return received, fmt.Errorf("stream timed out after %s", timeout)
		case <-ctx.Done():
			return received, ctx.Err()
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				if isWsClosed(err) {
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
