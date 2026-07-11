package sandbox0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxCreateAndListSessions(t *testing.T) {
	created := executionSessionFixture()
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sandboxes/sb_123/sessions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		switch r.Method {
		case http.MethodPost:
			if got := r.Header.Get("Idempotency-Key"); got != "create-1" {
				t.Fatalf("idempotency key = %q", got)
			}
			var request apispec.ExecutionSessionSpec
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if len(request.Command) != 2 || request.Command[1] != "hello" {
				t.Fatalf("command = %#v", request.Command)
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{"success": true, "data": created})
		case http.MethodGet:
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data":    map[string]any{"sessions": []any{created}},
			})
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	defer server.Close()

	sandbox := client.Sandbox("sb_123")
	value, err := sandbox.CreateSession(context.Background(), apispec.ExecutionSessionSpec{
		Command: []string{"/bin/echo", "hello"},
	}, &CreateSessionOptions{IdempotencyKey: "create-1"})
	if err != nil {
		t.Fatal(err)
	}
	if value.ID != "ses_123" {
		t.Fatalf("session ID = %q", value.ID)
	}
	values, err := sandbox.ListSessions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].ID != "ses_123" {
		t.Fatalf("sessions = %#v", values)
	}
}

func TestSandboxWatchSessionEventsResumesWithLastEventID(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sandboxes/sb_123/sessions/ses_123/events/stream" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("after"); got != "6" {
			t.Fatalf("after = %q", got)
		}
		if got := r.Header.Get("Last-Event-ID"); got != "7" {
			t.Fatalf("Last-Event-ID = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, ": heartbeat\n\nid: 8\nevent: output\ndata: %s\n\n", mustJSON(t, executionSessionEventFixture(8)))
	})
	defer server.Close()

	stream, err := client.Sandbox("sb_123").WatchSessionEvents(context.Background(), "ses_123", &SessionEventStreamOptions{
		After: 6, LastEventID: "7",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	event, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if event.Seq != 8 || event.Type != "output" {
		t.Fatalf("event = %#v", event)
	}
}

func executionSessionFixture() map[string]any {
	now := time.Unix(1_700_000_000, 0).UTC().Format(time.RFC3339)
	return map[string]any{
		"id":                 "ses_123",
		"spec":               map[string]any{"command": []string{"/bin/echo", "hello"}},
		"spec_version":       1,
		"phase":              "running",
		"runtime_generation": 3,
		"restart_count":      0,
		"cursor":             map[string]any{"earliest": 1, "latest": 7},
		"created_at":         now,
		"updated_at":         now,
		"last_activity_at":   now,
	}
}

func executionSessionEventFixture(seq int64) map[string]any {
	return map[string]any{
		"seq":                seq,
		"session_id":         "ses_123",
		"runtime_generation": 3,
		"attempt_id":         "att_1",
		"type":               "output",
		"stream":             "stdout",
		"data_base64":        "aGVsbG8=",
		"occurred_at":        time.Unix(1_700_000_000, 0).UTC().Format(time.RFC3339),
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSessionURLPreservesNumericCursor(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("after") != strconv.FormatInt(42, 10) {
			t.Fatalf("after = %q", r.URL.Query().Get("after"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()
	stream, err := client.Sandbox("sb_123").WatchSessionEvents(context.Background(), "ses_123", &SessionEventStreamOptions{After: 42})
	if err != nil {
		t.Fatal(err)
	}
	_ = stream.Close()
}
