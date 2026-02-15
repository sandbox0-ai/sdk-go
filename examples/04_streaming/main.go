package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create a client with auth (and optional base URL).
	client, err := sandbox0.NewClient(
		sandbox0.WithToken(os.Getenv("SANDBOX0_TOKEN")),
		sandbox0.WithBaseURL(os.Getenv("SANDBOX0_BASE_URL")),
	)
	must(err)

	// Claim a sandbox from a template and ensure cleanup.
	sandbox, err := client.ClaimSandbox(ctx, "default", sandbox0.WithSandboxHardTTL(300))
	must(err)
	defer client.DeleteSandbox(ctx, sandbox.ID)

	// Example 1: REPL stream using raw WebSocket
	fmt.Println("REPL stream:")
	must(runReplStream(ctx, sandbox))

	// Example 2: CMD stream using raw WebSocket
	fmt.Println("CMD stream:")
	must(runCmdStream(ctx, sandbox))
}

func runReplStream(ctx context.Context, sandbox *sandbox0.Sandbox) error {
	// 1. Create REPL context
	ctxResp, err := sandbox.CreateContext(ctx, apispec.CreateContextRequest{
		Type: apispec.NewOptProcessType(apispec.ProcessTypeRepl),
		Repl: apispec.NewOptCreateREPLContextRequest(apispec.CreateREPLContextRequest{
			Language: apispec.NewOptString("python"),
		}),
	})
	if err != nil {
		return err
	}

	// 2. Connect WebSocket
	conn, _, err := sandbox.ConnectWSContext(ctx, ctxResp.ID)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 3. Handle WebSocket read/write
	var wg sync.WaitGroup
	done := make(chan struct{})

	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if !isWsClosed(err) {
					log.Printf("read error: %v", err)
				}
				return
			}
			var msg struct {
				Source string `json:"source"`
				Data   string `json:"data"`
			}
			if err := json.Unmarshal(message, &msg); err != nil {
				continue
			}
			fmt.Print(msg.Data)
		}
	}()

	// Send inputs
	inputs := []string{
		"print('hello from repl')\n",
		"print(1 + 2)\n",
	}
	for _, input := range inputs {
		msg := map[string]any{
			"type":       "input",
			"data":       input,
			"request_id": fmt.Sprintf("req-%d", time.Now().UnixNano()),
		}
		if err := conn.WriteJSON(msg); err != nil {
			return err
		}
	}

	// Wait briefly then close
	time.Sleep(500 * time.Millisecond)
	close(done)
	conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	)

	wg.Wait()
	return nil
}

func runCmdStream(ctx context.Context, sandbox *sandbox0.Sandbox) error {
	// 1. Create CMD context (async, don't wait for completion)
	ctxResp, err := sandbox.CreateContext(ctx, apispec.CreateContextRequest{
		Type:          apispec.NewOptProcessType(apispec.ProcessTypeCmd),
		Cmd:           apispec.NewOptCreateCMDContextRequest(apispec.CreateCMDContextRequest{Command: []string{"bash", "-c", "for i in 1 2 3; do echo line-$i; done"}}),
		WaitUntilDone: apispec.NewOptBool(false),
	})
	if err != nil {
		return err
	}
	defer sandbox.DeleteContext(ctx, ctxResp.ID)

	// 2. Connect WebSocket
	conn, _, err := sandbox.ConnectWSContext(ctx, ctxResp.ID)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 3. Read outputs until stream closes
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if !isWsClosed(err) {
				return err
			}
			break
		}
		var msg struct {
			Source string `json:"source"`
			Data   string `json:"data"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}
		fmt.Print(msg.Data)
	}

	return nil
}

func isWsClosed(err error) bool {
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
