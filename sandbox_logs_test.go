package sandbox0

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestGetSandboxLogs(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/sandboxes/sb_123/logs" {
			t.Fatalf("path = %s, want /api/v1/sandboxes/sb_123/logs", r.URL.Path)
		}
		if got := r.URL.Query().Get("tail_lines"); got != "20" {
			t.Fatalf("tail_lines = %q, want 20", got)
		}
		if got := r.URL.Query().Get("timestamps"); got != "true" {
			t.Fatalf("timestamps = %q, want true", got)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Sandbox-ID", "sb_123")
		w.Header().Set("X-Sandbox-Pod-Name", "pod-a")
		w.Header().Set("X-Sandbox-Log-Container", "procd")
		w.Header().Set("X-Sandbox-Log-Previous", "false")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ready\n")
	})
	defer server.Close()

	tailLines := int64(20)
	logs, err := client.GetSandboxLogs(context.Background(), "sb_123", &SandboxLogsOptions{
		TailLines:  &tailLines,
		Timestamps: true,
	})
	if err != nil {
		t.Fatalf("GetSandboxLogs() error = %v", err)
	}
	if logs.Logs != "ready\n" || logs.Container != "procd" {
		t.Fatalf("logs = %+v, want procd ready output", logs)
	}
}

func TestStreamSandboxLogsReturnsLiveReader(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Query().Get("follow"); got != "true" {
			t.Fatalf("follow = %q, want true", got)
		}
		if got := r.URL.Query().Get("tail_lines"); got != "5" {
			t.Fatalf("tail_lines = %q, want 5", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Sandbox-ID", "sb_123")
		w.Header().Set("X-Sandbox-Pod-Name", "pod-a")
		w.Header().Set("X-Sandbox-Log-Container", "procd")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "line one")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tailLines := int64(5)
	stream, err := client.StreamSandboxLogs(ctx, "sb_123", &SandboxLogsOptions{TailLines: &tailLines})
	if err != nil {
		t.Fatalf("StreamSandboxLogs() error = %v", err)
	}
	defer stream.Close()

	if stream.SandboxID != "sb_123" || stream.PodName != "pod-a" || stream.Container != "procd" {
		t.Fatalf("stream metadata = %+v, want sandbox and pod headers", stream)
	}
	line, err := bufio.NewReader(stream).ReadString('\n')
	if err != nil {
		t.Fatalf("read stream line: %v", err)
	}
	if line != "line one\n" {
		t.Fatalf("line = %q, want line one", line)
	}
}
