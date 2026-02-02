package sandbox0

import (
	"context"
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestSandboxFileServiceSuccess(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("binary"))
	routes := routeMap{
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/files"): func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello"))
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/files/stat"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"name": "file.txt",
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/files/list"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"entries": []map[string]any{
						{"name": "file.txt"},
					},
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/files/binary"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"content": encoded,
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/files"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"written": true,
				},
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxes/sb-1/files"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"deleted": true,
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/files/move"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"moved": true,
				},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	sandbox := client.Sandbox("sb-1")
	ctx := context.Background()

	if data, err := sandbox.Files.Read(ctx, "/tmp/hello"); err != nil || string(data) != "hello" {
		t.Fatalf("read failed: %v, data=%q", err, data)
	}
	if _, err := sandbox.Files.Stat(ctx, "/tmp/hello"); err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if _, err := sandbox.Files.List(ctx, "/tmp"); err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if data, err := sandbox.Files.ReadBinary(ctx, "/tmp/bin"); err != nil || string(data) != "binary" {
		t.Fatalf("read binary failed: %v, data=%q", err, data)
	}
	if _, err := sandbox.Files.Write(ctx, "/tmp/hello", []byte("payload")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if _, err := sandbox.Files.Delete(ctx, "/tmp/hello"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := sandbox.Files.Move(ctx, "/tmp/hello", "/tmp/new"); err != nil {
		t.Fatalf("move failed: %v", err)
	}
}

func TestSandboxFileServiceMkdirAndWriteErrors(t *testing.T) {
	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/files"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"created": true,
				},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	sandbox := client.Sandbox("sb-1")
	ctx := context.Background()

	if _, err := sandbox.Files.Mkdir(ctx, "/tmp/dir", true); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	if _, err := sandbox.Files.Write(ctx, "/tmp/dir", []byte("data")); err == nil {
		t.Fatal("expected error when server returns created for write")
	}
}

func TestSandboxFileServiceWatch(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	routes := routeMap{
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/files/watch"): func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Fatalf("upgrade failed: %v", err)
			}
			defer conn.Close()

			var sub FileWatchSubscribeRequest
			if err := conn.ReadJSON(&sub); err != nil {
				t.Fatalf("read subscribe failed: %v", err)
			}
			if sub.Action != "subscribe" || sub.Path != "/tmp" || !sub.Recursive {
				t.Fatalf("unexpected subscribe: %#v", sub)
			}
			if err := conn.WriteJSON(FileWatchResponse{Type: "subscribed", WatchID: "watch-1"}); err != nil {
				t.Fatalf("write subscribe ack failed: %v", err)
			}
			if err := conn.WriteJSON(FileWatchResponse{Type: "event", Event: "write", Path: "/tmp/a"}); err != nil {
				t.Fatalf("write event failed: %v", err)
			}

			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			var unsub FileWatchUnsubscribeRequest
			_ = conn.ReadJSON(&unsub)
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	sandbox := client.Sandbox("sb-1")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events, errs, unsubscribe, err := sandbox.Files.Watch(ctx, "/tmp", true)
	if err != nil {
		t.Fatalf("watch failed: %v", err)
	}
	select {
	case ev := <-events:
		if ev.Path != "/tmp/a" || ev.Event != "write" {
			t.Fatalf("unexpected event: %#v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch event")
	}
	if err := unsubscribe(); err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}
	select {
	case <-errs:
	default:
	}
}
