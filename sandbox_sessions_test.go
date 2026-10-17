package sandbox0

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"strconv"
	"sync/atomic"
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

func TestConditionalSessionCreationNeverFallsBack(t *testing.T) {
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/sandboxes/sb_123/sessions/conditional" {
			t.Errorf("unconditional fallback: %s", r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") != "conditional-key" {
			t.Error("missing stable request identity")
		}
		var body apispec.ConditionalExecutionSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.SandboxID != "sb_123" || body.RuntimeGeneration != 7 {
			t.Errorf("lost launch condition: %+v", body)
		}
		if calls == 1 || calls == 3 {
			value := executionSessionFixture()
			if calls == 1 {
				value["runtime_generation"] = 7
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{"success": true, "data": value})
		} else {
			writeJSON(t, w, http.StatusNotFound, map[string]any{"success": false, "error": map[string]any{"code": "not_found", "message": "unsupported conditional route"}})
		}
	})
	defer server.Close()
	generation := int64(7)
	options := &CreateSessionOptions{IdempotencyKey: "conditional-key", RuntimeGeneration: &generation}
	spec := apispec.ExecutionSessionSpec{Command: []string{"/bin/true"}}
	if _, err := client.Sandbox("sb_123").CreateSession(context.Background(), spec, options); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Sandbox("sb_123").CreateSession(context.Background(), spec, options); err == nil {
		t.Fatal("unsupported route accepted")
	}
	if _, err := client.Sandbox("sb_123").CreateSession(context.Background(), spec, options); err == nil {
		t.Fatal("wrong-generation response accepted")
	}
	for _, invalid := range []int64{0, -1} {
		options.RuntimeGeneration = &invalid
		if _, err := client.Sandbox("sb_123").CreateSession(context.Background(), spec, options); err == nil {
			t.Fatal("invalid generation accepted")
		}
	}
	if calls != 3 {
		t.Fatalf("calls=%d", calls)
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

func TestFindConditionalSessionOnlyUsesReadOnlyRoute(t *testing.T) {
	calls := 0
	status, generation := 200, int64(3)
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/sandboxes/sb_123/sessions/conditional/lookup" || r.URL.Query().Get("runtime_generation") != "3" || r.Header.Get("Idempotency-Key") != "original-key" {
			t.Errorf("unexpected lookup %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Cache-Control", "no-store")
		if status != 200 {
			writeJSON(t, w, status, map[string]any{"success": false, "error": map[string]any{"code": "session_not_found", "message": "unavailable"}})
			return
		}
		fixture := executionSessionFixture()
		fixture["runtime_generation"] = generation
		writeJSON(t, w, 200, map[string]any{"success": true, "data": fixture})
	})
	defer server.Close()
	sandbox := client.Sandbox("sb_123")
	found, err := sandbox.FindConditionalSession(context.Background(), "original-key", 3)
	if err != nil || found.ID != "ses_123" {
		t.Fatalf("find: %v, %v", found, err)
	}
	for _, code := range []int{404, 405, 409, 503} {
		status = code
		if _, err := sandbox.FindConditionalSession(context.Background(), "original-key", 3); err == nil {
			t.Fatalf("status %d was adopted", code)
		}
	}
	status = 200
	generation = 4
	if _, err := sandbox.FindConditionalSession(context.Background(), "original-key", 3); err == nil {
		t.Fatal("wrong generation adopted")
	}
	for _, gen := range []int64{0, -1} {
		if _, err := sandbox.FindConditionalSession(context.Background(), "original-key", gen); err == nil {
			t.Fatal("bad generation accepted")
		}
	}
	if _, err := sandbox.FindConditionalSession(context.Background(), "", 3); err == nil {
		t.Fatal("empty key accepted")
	}
	if calls != 6 {
		t.Fatalf("lookup retried or fell back: %d", calls)
	}
}

func TestConnectConditionalSessionExactHandshakeAndNoFallback(t *testing.T) {
	var calls atomic.Int32
	var status atomic.Int32
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/api/v1/sandboxes/sb_123/sessions/session/conditional/ws" || r.Header.Get("Idempotency-Key") != "original" || r.URL.Query().Get("runtime_generation") != "7" || r.URL.Query().Get("attempt_id") != "attempt" || r.URL.Query().Get("after") != "5" {
			t.Errorf("incorrect attachment identity")
		}
		if code := status.Load(); code != 0 {
			w.WriteHeader(int(code))
			return
		}
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		if err := conn.WriteJSON(SessionWebSocketMessage{Type: "ack", RequestID: "fixture"}); err != nil {
			t.Error(err)
		}
	})
	defer server.Close()
	sb := client.Sandbox("sb_123")
	opts := &SessionWebSocketOptions{After: 5}
	conn, _, err := sb.ConnectConditionalSession(context.Background(), "session", "original", 7, "attempt", opts)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := conn.Recv()
	if err != nil || msg.RequestID != "fixture" {
		t.Fatalf("message=%v error=%v", msg, err)
	}
	conn.Close()
	for _, code := range []int32{404, 405, 409, 503} {
		status.Store(code)
		conn, resp, err := sb.ConnectConditionalSession(context.Background(), "session", "original", 7, "attempt", opts)
		if err == nil || conn != nil || resp.StatusCode != int(code) {
			t.Fatalf("status %d: %v", code, err)
		}
		resp.Body.Close()
	}
	for _, gen := range []int64{0, -1} {
		if _, _, err := sb.ConnectConditionalSession(context.Background(), "session", "original", gen, "attempt", opts); err == nil {
			t.Fatal("accepted invalid generation")
		}
	}
	for _, args := range [][3]string{{"", "original", "attempt"}, {"session", "", "attempt"}, {"session", "original", ""}} {
		if _, _, err := sb.ConnectConditionalSession(context.Background(), args[0], args[1], 7, args[2], opts); err == nil {
			t.Fatal("accepted missing identity")
		}
	}
	if _, _, err := sb.ConnectConditionalSession(context.Background(), "session", "original", 7, "attempt", &SessionWebSocketOptions{After: -1}); err == nil {
		t.Fatal("accepted negative cursor")
	}
	if calls.Load() != 5 {
		t.Fatalf("unexpected network requests: %d", calls.Load())
	}
}

func TestConnectRestoredConditionalSessionExactHandshakeAndNoFallback(t *testing.T) {
	var calls atomic.Int32
	var status atomic.Int32
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("creation_runtime_generation") != "1" || r.URL.Path != "/api/v1/sandboxes/sb_123/sessions/session/conditional/ws" || r.Header.Get("Idempotency-Key") != "original" || r.URL.Query().Get("runtime_generation") != "7" || r.URL.Query().Get("attempt_id") != "attempt" || r.URL.Query().Get("after") != "5" {
			t.Errorf("incorrect attachment identity")
		}
		if code := status.Load(); code != 0 {
			w.WriteHeader(int(code))
			return
		}
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		if err := conn.WriteJSON(SessionWebSocketMessage{Type: "ack", RequestID: "fixture"}); err != nil {
			t.Error(err)
		}
	})
	defer server.Close()
	sb := client.Sandbox("sb_123")
	opts := &SessionWebSocketOptions{After: 5}
	conn, _, err := sb.ConnectRestoredConditionalSession(context.Background(), "session", "original", 7, 1, "attempt", opts)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := conn.Recv()
	if err != nil || msg.RequestID != "fixture" {
		t.Fatalf("message=%v error=%v", msg, err)
	}
	conn.Close()
	for _, code := range []int32{404, 405, 409, 503} {
		status.Store(code)
		conn, resp, err := sb.ConnectRestoredConditionalSession(context.Background(), "session", "original", 7, 1, "attempt", opts)
		if err == nil || conn != nil || resp.StatusCode != int(code) {
			t.Fatalf("status %d: %v", code, err)
		}
		resp.Body.Close()
	}
	for _, gen := range []int64{0, -1} {
		if _, _, err := sb.ConnectRestoredConditionalSession(context.Background(), "session", "original", gen, 1, "attempt", opts); err == nil {
			t.Fatal("accepted invalid generation")
		}
	}
	for _, args := range [][3]string{{"", "original", "attempt"}, {"session", "", "attempt"}, {"session", "original", ""}} {
		if _, _, err := sb.ConnectRestoredConditionalSession(context.Background(), args[0], args[1], 7, 1, args[2], opts); err == nil {
			t.Fatal("accepted missing identity")
		}
	}
	if _, _, err := sb.ConnectRestoredConditionalSession(context.Background(), "session", "original", 7, 1, "attempt", &SessionWebSocketOptions{After: -1}); err == nil {
		t.Fatal("accepted negative cursor")
	}
	if calls.Load() != 5 {
		t.Fatalf("unexpected network requests: %d", calls.Load())
	}
}
