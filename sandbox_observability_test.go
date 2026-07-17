package sandbox0

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

const signedAuditEventID = "c48d73ec-a08f-41bb-82d2-3f48a827f9b2"

func TestListSandboxObservabilityEventsUsesV2Filters(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/sandboxes/sb_123/observability/events" {
			t.Fatalf("path = %s, want observability events path", r.URL.Path)
		}
		query := r.URL.Query()
		want := map[string]string{
			"source":        "cluster_gateway",
			"event_type":    "api_access",
			"outcome":       "succeeded",
			"actor_kind":    "api_key",
			"actor_id":      "key_1",
			"action":        "sandbox.read",
			"resource_type": "sandbox",
			"operation_id":  "op_123",
			"limit":         "25",
		}
		for key, value := range want {
			if got := query.Get(key); got != value {
				t.Fatalf("query %s = %q, want %q", key, got, value)
			}
		}
		writeSignedAuditResponse(t, w)
	})
	defer server.Close()

	response, err := client.Sandbox("sb_123").ListObservabilityEvents(
		context.Background(),
		&SandboxObservabilityEventOptions{
			SandboxObservabilityQueryOptions: SandboxObservabilityQueryOptions{Limit: 25},
			Source:                           apispec.ObservabilityEventSourceClusterGateway,
			EventType:                        apispec.SandboxObservabilityEventTypeAPIAccess,
			Outcome:                          apispec.SandboxObservabilityOutcomeSucceeded,
			ActorKind:                        apispec.SandboxAuditActorKindAPIKey,
			ActorID:                          "key_1",
			Action:                           "sandbox.read",
			ResourceType:                     "sandbox",
			OperationID:                      "op_123",
		},
	)
	if err != nil {
		t.Fatalf("ListObservabilityEvents() error = %v", err)
	}
	if len(response.Events) != 1 {
		t.Fatalf("events count = %d, want 1", len(response.Events))
	}
	event := response.Events[0]
	if got := event.EventID.String(); got != signedAuditEventID {
		t.Fatalf("event ID = %q, want %q", got, signedAuditEventID)
	}
	if event.Action != "sandbox.read" || event.Actor.Kind != apispec.SandboxAuditActorKindAPIKey {
		t.Fatalf("event = %+v, want signed sandbox.read API-key event", event)
	}
	if event.Integrity.SignatureStatus != apispec.SandboxAuditIntegritySignatureStatusVerified {
		t.Fatalf("signature status = %q, want verified", event.Integrity.SignatureStatus)
	}
}

func TestListSandboxObservabilityEventsSupportsExactEventID(t *testing.T) {
	eventID := uuid.MustParse(signedAuditEventID)
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("event_id"); got != eventID.String() {
			t.Fatalf("event_id = %q, want %q", got, eventID)
		}
		writeSignedAuditResponse(t, w)
	})
	defer server.Close()

	response, err := client.Sandbox("sb_123").ListObservabilityEvents(
		context.Background(),
		&SandboxObservabilityEventOptions{EventID: eventID},
	)
	if err != nil {
		t.Fatalf("ListObservabilityEvents() error = %v", err)
	}
	if len(response.Events) != 1 || response.Events[0].EventID != eventID {
		t.Fatalf("events = %+v, want exact event %s", response.Events, eventID)
	}
}

func TestWatchSandboxObservabilityEventsUsesV2Filters(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		want := map[string]string{
			"watch":         "true",
			"actor_kind":    "sandbox_workload",
			"actor_id":      "sb_123",
			"action":        "network.connect",
			"resource_type": "sandbox_network",
			"operation_id":  "op_456",
		}
		for key, value := range want {
			if got := query.Get(key); got != value {
				t.Fatalf("query %s = %q, want %q", key, got, value)
			}
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte("{\"type\":\"watermark\",\"cursor\":\"c1\",\"watermark\":\"2026-07-13T14:32:30Z\"}\n"))
	})
	defer server.Close()

	stream, err := client.Sandbox("sb_123").WatchObservabilityEvents(
		context.Background(),
		&SandboxObservabilityEventOptions{
			ActorKind:    apispec.SandboxAuditActorKindSandboxWorkload,
			ActorID:      "sb_123",
			Action:       "network.connect",
			ResourceType: "sandbox_network",
			OperationID:  "op_456",
		},
	)
	if err != nil {
		t.Fatalf("WatchObservabilityEvents() error = %v", err)
	}
	defer func() { _ = stream.Close() }()
	line, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv() error = %v", err)
	}
	if line.Cursor != "c1" || line.Watermark == "" {
		t.Fatalf("line = %+v, want watermark cursor", line)
	}
}

func writeSignedAuditResponse(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	writeJSON(t, w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"events": []map[string]any{{
				"event_id":       signedAuditEventID,
				"schema_version": 2,
				"team_id":        "team_1",
				"sandbox_id":     "sb_123",
				"region_id":      "region_1",
				"cluster_id":     "cluster_1",
				"occurred_at":    time.Date(2026, 7, 13, 14, 32, 29, 0, time.UTC),
				"ingested_at":    time.Date(2026, 7, 13, 14, 32, 30, 0, time.UTC),
				"source":         "cluster_gateway",
				"event_type":     "api_access",
				"phase":          "result",
				"outcome":        "succeeded",
				"actor":          map[string]any{"kind": "api_key", "id": "key_1"},
				"action":         "sandbox.read",
				"resource":       map[string]any{"type": "sandbox", "id": "sb_123"},
				"operation_id":   "op_123",
				"producer":       map[string]any{"service": "cluster-gateway"},
				"integrity": map[string]any{
					"algorithm":         "ed25519-sha256-v1",
					"payload_hash":      strings.Repeat("0", 64),
					"signature":         strings.Repeat("A", 86),
					"signing_key_id":    strings.Repeat("1", 64),
					"signature_status":  "verified",
					"event_id_conflict": false,
				},
				"attributes": map[string]any{},
			}},
			"next_cursor": "c2",
			"watermark":   "2026-07-13T14:32:30Z",
		},
	})
}
