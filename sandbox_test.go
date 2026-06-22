package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxCmdReturnsTerminalStatus(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/sandboxes/sb_test/contexts" {
			t.Fatalf("path = %s, want /api/v1/sandboxes/sb_test/contexts", r.URL.Path)
		}

		var req apispec.CreateContextRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		wait, ok := req.WaitUntilDone.Get()
		if !ok || !wait {
			t.Fatalf("wait_until_done = %v set=%v, want true", wait, ok)
		}

		writeJSON(t, w, http.StatusCreated, map[string]any{
			"success": true,
			"data": map[string]any{
				"id":         "ctx_cmd",
				"type":       "cmd",
				"running":    false,
				"paused":     false,
				"created_at": "2026-06-18T00:00:00Z",
				"output_raw": "outerr",
				"stdout":     "out",
				"stderr":     "err",
				"exit_code":  7,
				"state":      "crashed",
			},
		})
	})
	defer server.Close()

	sandbox := &Sandbox{ID: "sb_test", client: client}
	result, err := sandbox.Cmd(context.Background(), "/bin/sh -c 'exit 7'")
	if err != nil {
		t.Fatalf("Cmd() error = %v", err)
	}
	if result.OutputRaw != "outerr" {
		t.Fatalf("OutputRaw = %q, want outerr", result.OutputRaw)
	}
	if result.Stdout != "out" {
		t.Fatalf("Stdout = %q, want out", result.Stdout)
	}
	if result.Stderr != "err" {
		t.Fatalf("Stderr = %q, want err", result.Stderr)
	}
	if result.ExitCode == nil || *result.ExitCode != 7 {
		t.Fatalf("ExitCode = %v, want 7", result.ExitCode)
	}
	if result.State != "crashed" {
		t.Fatalf("State = %q, want crashed", result.State)
	}
}
