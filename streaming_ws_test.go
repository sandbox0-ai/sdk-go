package sandbox0

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSandboxRunStream(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	var gotInput atomic.Bool
	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         "ctx-1",
					"created_at": "2024-01-02T00:00:00Z",
					"paused":     false,
					"running":    true,
					"type":       "repl",
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/contexts/ctx-1/ws"): func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Fatalf("upgrade failed: %v", err)
			}
			defer conn.Close()

			var msg StreamInput
			if err := conn.ReadJSON(&msg); err != nil {
				t.Fatalf("read input failed: %v", err)
			}
			if msg.Type != StreamInputTypeInput || msg.RequestID == "" {
				t.Fatalf("unexpected input: %#v", msg)
			}
			gotInput.Store(true)
			if err := conn.WriteJSON(streamOutputMessage{Source: "stdout", Data: "ok"}); err != nil {
				t.Fatalf("write output failed: %v", err)
			}
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	sandbox := client.Sandbox("sb-1")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	input := make(chan StreamInput, 1)
	input <- StreamInput{Data: "hello"}
	close(input)

	outputs, errs, closeFn, err := sandbox.RunStream(ctx, "", input)
	if err != nil {
		t.Fatalf("run stream failed: %v", err)
	}
	select {
	case out := <-outputs:
		if out.Data != "ok" || out.Source != "stdout" {
			t.Fatalf("unexpected output: %#v", out)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for output")
	}
	_ = closeFn()
	if gotInput.Load() == false {
		t.Fatal("expected input to be received by server")
	}
	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("unexpected stream error: %v", err)
		}
	default:
	}
}

func TestSandboxCmdStreamCleanup(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	var deleteCount atomic.Int32
	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         "ctx-cmd",
					"created_at": "2024-01-02T00:00:00Z",
					"paused":     false,
					"running":    true,
					"type":       "cmd",
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/contexts/ctx-cmd/ws"): func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Fatalf("upgrade failed: %v", err)
			}
			defer conn.Close()
			if err := conn.WriteJSON(streamOutputMessage{Source: "stdout", Data: "done"}); err != nil {
				t.Fatalf("write output failed: %v", err)
			}
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxes/sb-1/contexts/ctx-cmd"): func(w http.ResponseWriter, r *http.Request) {
			deleteCount.Add(1)
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"deleted": true,
				},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	sandbox := client.Sandbox("sb-1")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	outputs, _, closeFn, err := sandbox.CmdStream(ctx, "echo hi", nil)
	if err != nil {
		t.Fatalf("cmd stream failed: %v", err)
	}
	select {
	case out := <-outputs:
		if out.Data != "done" {
			t.Fatalf("unexpected output: %#v", out)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for output")
	}
	_ = closeFn()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if deleteCount.Load() == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected cleanup delete to be called")
}
