package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

const signedAuditEventID = "c48d73ec-a08f-41bb-82d2-3f48a827f9b2"
const legacyAuditEventID = "ec51a762-3370-4d06-957d-3b6a57d2f12f"

func TestListSandboxObservabilityEventsUsesFilters(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/sandboxes/sb_123/observability/events" {
			t.Fatalf("path = %s, want observability events path", r.URL.Path)
		}
		query := r.URL.Query()
		want := map[string]string{
			"source":                      "netd",
			"event_type":                  "network_audit",
			"outcome":                     "completed",
			"actor_kind":                  "sandbox_workload",
			"actor_id":                    "sb_123",
			"execution_scope_namespace":   "codex",
			"execution_scope_kind":        "native_session",
			"execution_scope_id":          "thread_123",
			"execution_scope_attribution": "process_environment",
			"action":                      "network.connect",
			"resource_type":               "sandbox_network",
			"operation_id":                "op_123",
			"limit":                       "25",
			"max_schema_version":          "3",
		}
		for key, value := range want {
			if got := query.Get(key); got != value {
				t.Fatalf("query %s = %q, want %q", key, got, value)
			}
		}
		writeSignedAuditResponse(t, w, true)
	})
	defer server.Close()

	response, err := client.Sandbox("sb_123").ListObservabilityEvents(
		context.Background(),
		&SandboxObservabilityEventOptions{
			SandboxObservabilityQueryOptions: SandboxObservabilityQueryOptions{Limit: 25},
			Source:                           apispec.ObservabilityEventSourceNetd,
			EventType:                        apispec.SandboxObservabilityEventTypeNetworkAudit,
			Outcome:                          apispec.SandboxObservabilityOutcomeCompleted,
			ActorKind:                        apispec.SandboxAuditActorKindSandboxWorkload,
			ActorID:                          "sb_123",
			ExecutionScopeNamespace:          "codex",
			ExecutionScopeKind:               "native_session",
			ExecutionScopeID:                 "thread_123",
			ExecutionScopeAttribution:        apispec.SandboxAuditExecutionScopeAttributionProcessEnvironment,
			Action:                           "network.connect",
			ResourceType:                     "sandbox_network",
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
	if event.Action != "network.connect" || event.Actor.Kind != apispec.SandboxAuditActorKindSandboxWorkload {
		t.Fatalf("event = %+v, want scoped sandbox-workload network event", event)
	}
	if event.Integrity.SignatureStatus != apispec.SandboxAuditIntegritySignatureStatusVerified {
		t.Fatalf("signature status = %q, want verified", event.Integrity.SignatureStatus)
	}
	if response.EffectiveQuery.MaxSchemaVersion != currentSandboxObservabilityEventSchemaVersion {
		t.Fatalf("effective max_schema_version = %d, want %d", response.EffectiveQuery.MaxSchemaVersion, currentSandboxObservabilityEventSchemaVersion)
	}
	if scopeID, ok := response.EffectiveQuery.ExecutionScopeID.Get(); !ok || scopeID != "thread_123" {
		t.Fatalf("effective execution_scope_id = %q, set = %v", scopeID, ok)
	}
	scope, ok := event.ExecutionScope.Get()
	if !ok {
		t.Fatal("execution scope is not set")
	}
	if scope.Namespace != "codex" ||
		scope.Kind != "native_session" ||
		scope.ID != "thread_123" ||
		scope.Attribution != apispec.SandboxAuditExecutionScopeAttributionProcessEnvironment {
		t.Fatalf("execution scope = %+v, want codex native session thread_123", scope)
	}
	roundTripEvent(t, event, true)
}

func TestListSandboxObservabilityEventsDecodesV2WithoutExecutionScope(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeLegacyAuditResponse(t, w)
	})
	defer server.Close()

	response, err := client.Sandbox("sb_123").ListObservabilityEvents(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListObservabilityEvents() error = %v", err)
	}
	if len(response.Events) != 1 {
		t.Fatalf("events count = %d, want 1", len(response.Events))
	}
	event := response.Events[0]
	if event.EventID.String() != legacyAuditEventID {
		t.Fatalf("event ID = %q, want %q", event.EventID, legacyAuditEventID)
	}
	if event.SchemaVersion != apispec.SandboxObservabilityEventSchemaVersion2 {
		t.Fatalf("schema version = %d, want 2", event.SchemaVersion)
	}
	if event.ExecutionScope.IsSet() {
		t.Fatalf("execution scope = %+v, want unset for v2 event", event.ExecutionScope)
	}
	roundTripEvent(t, event, false)
}

func TestListSandboxObservabilityEventsSupportsExactEventID(t *testing.T) {
	eventID := uuid.MustParse(signedAuditEventID)
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("event_id"); got != eventID.String() {
			t.Fatalf("event_id = %q, want %q", got, eventID)
		}
		if got := r.URL.Query().Get("max_schema_version"); got != "3" {
			t.Fatalf("max_schema_version = %q, want 3", got)
		}
		writeSignedAuditResponse(t, w, false)
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

func TestListSandboxObservabilityEventsSupportsLegacySchemaOptOut(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("max_schema_version"); got != "2" {
			t.Fatalf("max_schema_version = %q, want 2", got)
		}
		writeLegacyAuditResponse(t, w)
	})
	defer server.Close()

	response, err := client.Sandbox("sb_123").ListObservabilityEvents(
		context.Background(),
		&SandboxObservabilityEventOptions{MaxSchemaVersion: 2},
	)
	if err != nil {
		t.Fatalf("ListObservabilityEvents() error = %v", err)
	}
	if len(response.Events) != 1 ||
		response.Events[0].SchemaVersion != apispec.SandboxObservabilityEventSchemaVersion2 {
		t.Fatalf("events = %+v, want one v2 event", response.Events)
	}
}

func TestSandboxObservabilityExecutionScopeRejectsLegacySchemaOptOut(t *testing.T) {
	tests := []struct {
		name string
		call func(*Client) error
	}{
		{
			name: "list",
			call: func(client *Client) error {
				_, err := client.Sandbox("sb_123").ListObservabilityEvents(
					context.Background(),
					&SandboxObservabilityEventOptions{
						MaxSchemaVersion:        2,
						ExecutionScopeNamespace: "codex",
					},
				)
				return err
			},
		},
		{
			name: "watch",
			call: func(client *Client) error {
				_, err := client.Sandbox("sb_123").WatchObservabilityEvents(
					context.Background(),
					&SandboxObservabilityEventOptions{
						MaxSchemaVersion: 2,
						ExecutionScopeID: "thread_123",
					},
				)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requested := false
			client, server := newTestClient(t, func(http.ResponseWriter, *http.Request) {
				requested = true
			})
			defer server.Close()

			err := tt.call(client)
			if err == nil || !strings.Contains(err.Error(), "max_schema_version >= 3") {
				t.Fatalf("call error = %v, want max_schema_version compatibility error", err)
			}
			if requested {
				t.Fatal("call sent a request for incompatible execution scope filters")
			}
		})
	}
}

func TestWatchSandboxObservabilityEventsRejectsUnsupportedOptions(t *testing.T) {
	endTime := time.Now()
	tests := []struct {
		name      string
		options   *SandboxObservabilityEventOptions
		wantError string
	}{
		{
			name: "end time",
			options: &SandboxObservabilityEventOptions{
				SandboxObservabilityQueryOptions: SandboxObservabilityQueryOptions{EndTime: &endTime},
			},
			wantError: "end_time",
		},
		{
			name:      "event ID",
			options:   &SandboxObservabilityEventOptions{EventID: uuid.MustParse(signedAuditEventID)},
			wantError: "event_id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requested := false
			client, server := newTestClient(t, func(http.ResponseWriter, *http.Request) {
				requested = true
			})
			defer server.Close()

			stream, err := client.Sandbox("sb_123").WatchObservabilityEvents(context.Background(), tt.options)
			if stream != nil {
				_ = stream.Close()
				t.Fatal("WatchObservabilityEvents() returned a stream for invalid options")
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("WatchObservabilityEvents() error = %v, want error containing %q", err, tt.wantError)
			}
			if requested {
				t.Fatal("WatchObservabilityEvents() sent a request for invalid options")
			}
		})
	}
}

func TestWatchSandboxObservabilityLogsRejectsEndTime(t *testing.T) {
	endTime := time.Now()
	requested := false
	client, server := newTestClient(t, func(http.ResponseWriter, *http.Request) {
		requested = true
	})
	defer server.Close()

	stream, err := client.Sandbox("sb_123").WatchLogs(
		context.Background(),
		&SandboxObservabilityLogOptions{
			SandboxObservabilityQueryOptions: SandboxObservabilityQueryOptions{EndTime: &endTime},
		},
	)
	if stream != nil {
		_ = stream.Close()
		t.Fatal("WatchLogs() returned a stream for invalid options")
	}
	if err == nil || !strings.Contains(err.Error(), "end_time") {
		t.Fatalf("WatchLogs() error = %v, want error containing %q", err, "end_time")
	}
	if requested {
		t.Fatal("WatchLogs() sent a request for invalid options")
	}
}

func TestWatchSandboxObservabilityEventsUsesFilters(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		want := map[string]string{
			"watch":                       "true",
			"actor_kind":                  "sandbox_workload",
			"actor_id":                    "sb_123",
			"execution_scope_namespace":   "codex",
			"execution_scope_kind":        "native_session",
			"execution_scope_id":          "thread_456",
			"execution_scope_attribution": "process_environment",
			"action":                      "network.connect",
			"resource_type":               "sandbox_network",
			"operation_id":                "op_456",
			"max_schema_version":          "3",
		}
		for key, value := range want {
			if got := query.Get(key); got != value {
				t.Fatalf("query %s = %q, want %q", key, got, value)
			}
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(
			"{\"type\":\"ready\",\"effective_query\":{\"max_schema_version\":3,\"execution_scope_namespace\":\"codex\",\"execution_scope_kind\":\"native_session\",\"execution_scope_id\":\"thread_456\",\"execution_scope_attribution\":\"process_environment\"}}\n" +
				"{\"type\":\"watermark\",\"cursor\":\"c1\",\"watermark\":\"2026-07-13T14:32:30Z\"}\n",
		))
	})
	defer server.Close()

	stream, err := client.Sandbox("sb_123").WatchObservabilityEvents(
		context.Background(),
		&SandboxObservabilityEventOptions{
			ActorKind:                 apispec.SandboxAuditActorKindSandboxWorkload,
			ActorID:                   "sb_123",
			ExecutionScopeNamespace:   "codex",
			ExecutionScopeKind:        "native_session",
			ExecutionScopeID:          "thread_456",
			ExecutionScopeAttribution: apispec.SandboxAuditExecutionScopeAttributionProcessEnvironment,
			Action:                    "network.connect",
			ResourceType:              "sandbox_network",
			OperationID:               "op_456",
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
	if line.Type != "ready" || line.EffectiveQuery == nil {
		t.Fatalf("first line = %+v, want ready with effective query", line)
	}
	if line.EffectiveQuery.MaxSchemaVersion != currentSandboxObservabilityEventSchemaVersion {
		t.Fatalf("ready max_schema_version = %d, want %d", line.EffectiveQuery.MaxSchemaVersion, currentSandboxObservabilityEventSchemaVersion)
	}
	if scopeID, ok := line.EffectiveQuery.ExecutionScopeID.Get(); !ok || scopeID != "thread_456" {
		t.Fatalf("ready execution_scope_id = %q, set = %v", scopeID, ok)
	}
	line, err = stream.Recv()
	if err != nil {
		t.Fatalf("Recv() watermark error = %v", err)
	}
	if line.Cursor != "c1" || line.Watermark == "" {
		t.Fatalf("line = %+v, want watermark cursor", line)
	}
}

func TestExecutionSessionScopeSpecJSONRoundTrip(t *testing.T) {
	spec := apispec.ExecutionSessionSpec{
		Command: []string{"codex", "app-server"},
		ExecutionScope: apispec.NewOptExecutionSessionScopeSpec(
			apispec.ExecutionSessionScopeSpec{
				Namespace:             "codex",
				Kind:                  "native_session",
				IDEnvironmentVariable: "CODEX_THREAD_ID",
			},
		),
	}
	encoded, err := json.Marshal(&spec)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if !strings.Contains(string(encoded), `"execution_scope":{"namespace":"codex","kind":"native_session","id_environment_variable":"CODEX_THREAD_ID"}`) {
		t.Fatalf("encoded spec = %s, want execution_scope", encoded)
	}
	var decoded apispec.ExecutionSessionSpec
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	scope, ok := decoded.ExecutionScope.Get()
	if !ok || scope.IDEnvironmentVariable != "CODEX_THREAD_ID" {
		t.Fatalf("decoded scope = %+v, set = %v", scope, ok)
	}
}

func writeSignedAuditResponse(t *testing.T, w http.ResponseWriter, scopedEffectiveQuery bool) {
	t.Helper()
	event := auditEventFixture(signedAuditEventID, apispec.SandboxObservabilityEventSchemaVersion3)
	event["execution_scope"] = map[string]any{
		"namespace":   "codex",
		"kind":        "native_session",
		"id":          "thread_123",
		"attribution": "process_environment",
	}
	effectiveQuery := map[string]any{"max_schema_version": 3}
	if scopedEffectiveQuery {
		effectiveQuery["execution_scope_namespace"] = "codex"
		effectiveQuery["execution_scope_kind"] = "native_session"
		effectiveQuery["execution_scope_id"] = "thread_123"
		effectiveQuery["execution_scope_attribution"] = "process_environment"
	}
	writeAuditResponse(t, w, event, effectiveQuery)
}

func writeLegacyAuditResponse(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	writeAuditResponse(
		t,
		w,
		auditEventFixture(legacyAuditEventID, apispec.SandboxObservabilityEventSchemaVersion2),
		map[string]any{"max_schema_version": 2},
	)
}

func writeAuditResponse(t *testing.T, w http.ResponseWriter, event, effectiveQuery map[string]any) {
	t.Helper()
	writeJSON(t, w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"events":          []map[string]any{event},
			"next_cursor":     "c2",
			"watermark":       "2026-07-13T14:32:30Z",
			"effective_query": effectiveQuery,
		},
	})
}

func auditEventFixture(eventID string, schemaVersion apispec.SandboxObservabilityEventSchemaVersion) map[string]any {
	return map[string]any{
		"event_id":       eventID,
		"schema_version": schemaVersion,
		"team_id":        "team_1",
		"sandbox_id":     "sb_123",
		"region_id":      "region_1",
		"cluster_id":     "cluster_1",
		"occurred_at":    time.Date(2026, 7, 13, 14, 32, 29, 0, time.UTC),
		"ingested_at":    time.Date(2026, 7, 13, 14, 32, 30, 0, time.UTC),
		"source":         "netd",
		"event_type":     "network_audit",
		"phase":          "result",
		"outcome":        "completed",
		"actor":          map[string]any{"kind": "sandbox_workload", "id": "sb_123"},
		"action":         "network.connect",
		"resource":       map[string]any{"type": "sandbox_network", "id": "sb_123"},
		"operation_id":   "op_123",
		"producer":       map[string]any{"service": "netd"},
		"integrity": map[string]any{
			"algorithm":         "ed25519-sha256-v1",
			"payload_hash":      strings.Repeat("0", 64),
			"signature":         strings.Repeat("A", 86),
			"signing_key_id":    strings.Repeat("1", 64),
			"signature_status":  "verified",
			"event_id_conflict": false,
		},
		"attributes": map[string]any{},
	}
}

func roundTripEvent(t *testing.T, event apispec.SandboxObservabilityEvent, wantScope bool) {
	t.Helper()
	encoded, err := json.Marshal(&event)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded apispec.SandboxObservabilityEvent
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.SchemaVersion != event.SchemaVersion {
		t.Fatalf("round-trip schema version = %d, want %d", decoded.SchemaVersion, event.SchemaVersion)
	}
	if decoded.ExecutionScope.IsSet() != wantScope {
		t.Fatalf("round-trip execution scope set = %v, want %v", decoded.ExecutionScope.IsSet(), wantScope)
	}
}
