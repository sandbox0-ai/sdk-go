package sandbox0

import (
	"context"
	"net/http"
	"testing"
)

func TestClientListObservabilityTraceSpans(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/observability/traces" {
			t.Fatalf("path = %s, want /api/v1/observability/traces", r.URL.Path)
		}
		if got := r.URL.Query().Get("sandbox_id"); got != "sb_123" {
			t.Fatalf("sandbox_id = %q, want sb_123", got)
		}
		if got := r.URL.Query().Get("trace_id"); got != "tr_123" {
			t.Fatalf("trace_id = %q, want tr_123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"spans":[{"timestamp":"2026-05-07 12:00:00","trace_id":"tr_123","span_id":"sp_1","name":"HTTP GET /api/v1/sandboxes/{id}","duration_nano":42,"resource_attributes":{"sandbox0.sandbox_id":"sb_123"}}]}}`))
	})
	defer server.Close()

	spans, err := client.ListObservabilityTraceSpans(context.Background(), &ObservabilityQueryOptions{
		SandboxID: "sb_123",
		TraceID:   "tr_123",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListObservabilityTraceSpans() error = %v", err)
	}
	if len(spans) != 1 || spans[0].TraceID.Or("") != "tr_123" || spans[0].SpanID.Or("") != "sp_1" {
		t.Fatalf("unexpected spans: %#v", spans)
	}
}

func TestClientListObservabilityLogs(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/observability/logs" {
			t.Fatalf("path = %s, want /api/v1/observability/logs", r.URL.Path)
		}
		if got := r.URL.Query().Get("sandbox_id"); got != "sb_123" {
			t.Fatalf("sandbox_id = %q, want sb_123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"logs":[{"timestamp":"2026-05-07 12:00:00","trace_id":"tr_123","span_id":"sp_1","severity_text":"INFO","body":"ready"}]}}`))
	})
	defer server.Close()

	logs, err := client.ListObservabilityLogs(context.Background(), &ObservabilityQueryOptions{
		SandboxID: "sb_123",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListObservabilityLogs() error = %v", err)
	}
	if len(logs) != 1 || logs[0].Body.Or("") != "ready" || logs[0].SeverityText.Or("") != "INFO" {
		t.Fatalf("unexpected logs: %#v", logs)
	}
}
