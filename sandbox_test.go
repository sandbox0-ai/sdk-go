package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestParseCommand(t *testing.T) {
	args, err := parseCommand(`python -c "print('ok')"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 3 || args[0] != "python" {
		t.Fatalf("unexpected args: %#v", args)
	}
	if _, err := parseCommand(" "); err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestSandboxRunUsesCachedReplContext(t *testing.T) {
	var createCount atomic.Int32
	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts"): func(w http.ResponseWriter, r *http.Request) {
			createCount.Add(1)
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			repl, ok := payload["repl"].(map[string]any)
			if !ok || repl["language"] != "python" {
				t.Fatalf("expected repl language python, got %#v", payload["repl"])
			}
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
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts/ctx-1/exec"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"output": "ok",
				},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	sandbox := client.Sandbox("sb-1")
	ctx := context.Background()

	if _, err := sandbox.Run(ctx, "", "print('a')"); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if _, err := sandbox.Run(ctx, "python", "print('b')"); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if got := createCount.Load(); got != 1 {
		t.Fatalf("expected 1 context create, got %d", got)
	}
}

func TestSandboxCmdCreatesAndDeletesContext(t *testing.T) {
	var deleteCount atomic.Int32
	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         "ctx-cmd",
					"created_at": "2024-01-02T00:00:00Z",
					"paused":     false,
					"running":    false,
					"type":       "cmd",
					"output":     "done",
				},
			})
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
	ctx := context.Background()

	result, err := sandbox.Cmd(ctx, "echo hello")
	if err != nil {
		t.Fatalf("cmd failed: %v", err)
	}
	if result.Output != "done" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
	if got := deleteCount.Load(); got != 1 {
		t.Fatalf("expected delete called once, got %d", got)
	}
	if _, err := sandbox.Cmd(ctx, " "); err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestSandboxRunWithContextIDSkipsCreate(t *testing.T) {
	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts/ctx-direct/exec"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"output": "ok",
				},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	sandbox := client.Sandbox("sb-1")
	ctx := context.Background()

	if _, err := sandbox.Run(ctx, "python", "print('x')", WithContextID("ctx-direct")); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if _, err := sandbox.Run(ctx, "python", " "); err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestSandboxRunOptionsApplied(t *testing.T) {
	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts"): func(w http.ResponseWriter, r *http.Request) {
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if payload["cwd"] != "/work" {
				t.Fatalf("unexpected cwd: %#v", payload["cwd"])
			}
			envVars, ok := payload["env_vars"].(map[string]any)
			if !ok || envVars["A"] != "B" {
				t.Fatalf("unexpected env vars: %#v", payload["env_vars"])
			}
			if int(payload["idle_timeout_sec"].(float64)) != 5 {
				t.Fatalf("unexpected idle timeout: %#v", payload["idle_timeout_sec"])
			}
			if int(payload["ttl_sec"].(float64)) != 10 {
				t.Fatalf("unexpected ttl: %#v", payload["ttl_sec"])
			}
			pty, ok := payload["pty_size"].(map[string]any)
			if !ok || int(pty["rows"].(float64)) != 24 || int(pty["cols"].(float64)) != 80 {
				t.Fatalf("unexpected pty size: %#v", payload["pty_size"])
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         "ctx-opts",
					"created_at": "2024-01-02T00:00:00Z",
					"paused":     false,
					"running":    true,
					"type":       "repl",
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts/ctx-opts/exec"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"output": "ok",
				},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	sandbox := client.Sandbox("sb-1")
	ctx := context.Background()

	_, err := sandbox.Run(
		ctx,
		"python",
		"print('x')",
		WithCWD("/work"),
		WithEnvVars(map[string]string{"A": "B"}),
		WithIdleTimeout(5),
		WithContextTTL(10),
		WithPTYSize(24, 80),
	)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestSandboxCmdOptionsApplied(t *testing.T) {
	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/contexts"): func(w http.ResponseWriter, r *http.Request) {
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			cmd, ok := payload["cmd"].(map[string]any)
			if !ok {
				t.Fatalf("missing cmd payload: %#v", payload["cmd"])
			}
			command := cmd["command"].([]any)
			if len(command) != 2 || command[0] != "echo" {
				t.Fatalf("unexpected command: %#v", cmd["command"])
			}
			if payload["cwd"] != "/work" {
				t.Fatalf("unexpected cwd: %#v", payload["cwd"])
			}
			if int(payload["idle_timeout_sec"].(float64)) != 5 {
				t.Fatalf("unexpected idle timeout: %#v", payload["idle_timeout_sec"])
			}
			if int(payload["ttl_sec"].(float64)) != 10 {
				t.Fatalf("unexpected ttl: %#v", payload["ttl_sec"])
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":         "ctx-cmd",
					"created_at": "2024-01-02T00:00:00Z",
					"paused":     false,
					"running":    false,
					"type":       "cmd",
					"output":     "ok",
				},
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxes/sb-1/contexts/ctx-cmd"): func(w http.ResponseWriter, r *http.Request) {
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
	ctx := context.Background()

	_, err := sandbox.Cmd(
		ctx,
		"ignored",
		WithCommand([]string{"echo", "hi"}),
		WithCmdCWD("/work"),
		WithCmdIdleTimeout(5),
		WithCmdTTL(10),
		WithCmdPTYSize(24, 80),
	)
	if err != nil {
		t.Fatalf("cmd failed: %v", err)
	}
}
