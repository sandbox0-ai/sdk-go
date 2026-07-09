package sandbox0

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSandboxProcessesCreateAndSendInput(t *testing.T) {
	t.Parallel()

	var createBody map[string]any
	var inputBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/sb_123/processes":
			mustDecodeJSON(t, r.Body, &createBody)
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"success":true,"data":{"process":{"id":"proc_1","command":["python","-u","-c","print(1)"],"state":"running","created_at":"2026-07-09T00:00:00Z","channels":[{"name":"stdio","kind":"stdio","framing":"line","stdin":true,"stdout":true,"stderr":true}],"event_log":{"next_seq":1,"oldest_seq":1,"capacity":1024}}}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/sandboxes/sb_123/processes/proc_1/events":
			mustDecodeJSON(t, r.Body, &inputBody)
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"success":true,"data":{"event":{"seq":2,"event_id":"evt_1","process_id":"proc_1","channel":"stdio","type":"stdin.write","timestamp":"2026-07-09T00:00:01Z","payload":{"data":"hello\n"}}}}`)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(WithBaseURL(server.URL), WithToken("test-token"))
	if err != nil {
		t.Fatal(err)
	}
	sandbox := client.Sandbox("sb_123")

	spec := StdioProcessSpec([]string{"python", "-u", "-c", "print(1)"})
	process, err := sandbox.CreateProcess(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if process.ID != "proc_1" {
		t.Fatalf("process ID = %q, want proc_1", process.ID)
	}
	if got := createBody["command"]; !deepEqualJSON(got, []any{"python", "-u", "-c", "print(1)"}) {
		t.Fatalf("create command = %#v", got)
	}

	event, err := NewProcessStdinEvent("evt_1", "stdio", "hello\n")
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := sandbox.SendProcessEvent(context.Background(), "proc_1", event)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Seq != 2 {
		t.Fatalf("accepted seq = %d, want 2", accepted.Seq)
	}
	if got := inputBody["event_id"]; got != "evt_1" {
		t.Fatalf("event_id = %#v, want evt_1", got)
	}
	if payload, ok := inputBody["payload"].(map[string]any); !ok || payload["data"] != "hello\n" {
		t.Fatalf("payload = %#v, want data", inputBody["payload"])
	}
}

func TestSandboxWatchProcessEventsCanResumeFromCursor(t *testing.T) {
	t.Parallel()

	requests := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/sandboxes/sb_123/processes/proc_1/events" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		requests = append(requests, r.URL.RawQuery)
		w.Header().Set("Content-Type", "text/event-stream")
		if r.URL.Query().Get("cursor") == "1" {
			_, _ = io.WriteString(w, "data: {\"seq\":2,\"process_id\":\"proc_1\",\"channel\":\"stdio\",\"type\":\"stdout.line\",\"timestamp\":\"2026-07-09T00:00:02Z\",\"payload\":{\"data\":\"second\"}}\n\n")
			return
		}
		_, _ = io.WriteString(w, "data: {\"seq\":1,\"process_id\":\"proc_1\",\"channel\":\"stdio\",\"type\":\"stdout.line\",\"timestamp\":\"2026-07-09T00:00:01Z\",\"payload\":{\"data\":\"first\"}}\n\n")
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(WithBaseURL(server.URL), WithToken("test-token"))
	if err != nil {
		t.Fatal(err)
	}
	sandbox := client.Sandbox("sb_123")

	first, err := sandbox.WatchProcessEvents(context.Background(), "proc_1", nil)
	if err != nil {
		t.Fatal(err)
	}
	event, err := first.Recv()
	_ = first.Close()
	if err != nil {
		t.Fatal(err)
	}
	if event.Seq != 1 {
		t.Fatalf("first seq = %d, want 1", event.Seq)
	}

	cursor := event.Seq
	second, err := sandbox.WatchProcessEvents(context.Background(), "proc_1", &ProcessEventWatchOptions{Cursor: &cursor})
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := second.Recv()
	_ = second.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Seq != 2 {
		t.Fatalf("resumed seq = %d, want 2", resumed.Seq)
	}
	if len(requests) != 2 || requests[1] != "cursor=1" {
		t.Fatalf("requests = %#v, want second cursor=1", requests)
	}
}

func mustDecodeJSON(t *testing.T, r io.Reader, dst any) {
	t.Helper()
	if err := json.NewDecoder(r).Decode(dst); err != nil {
		t.Fatal(err)
	}
}

func deepEqualJSON(got any, want any) bool {
	gotBytes, _ := json.Marshal(got)
	wantBytes, _ := json.Marshal(want)
	return strings.EqualFold(string(gotBytes), string(wantBytes))
}
