package sandbox0

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestCreateTemplateFromSandbox(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/templates/from-sandbox" {
			t.Fatalf("path = %s, want /api/v1/templates/from-sandbox", r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "build-1" {
			t.Fatalf("Idempotency-Key = %q, want build-1", got)
		}
		var request apispec.TemplateFromSandboxCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.TemplateID != "python-ready" || request.SandboxID != "sb_source" {
			t.Fatalf("request = %+v", request)
		}
		writeTemplateResponse(t, w, http.StatusAccepted, "python-ready", "creating", "capturing")
	})
	defer server.Close()

	tpl, err := client.CreateTemplateFromSandbox(
		context.Background(),
		NewTemplateFromSandboxCreateRequest("python-ready", "sb_source", nil),
		&CreateTemplateFromSandboxOptions{IdempotencyKey: "build-1"},
	)
	if err != nil {
		t.Fatalf("CreateTemplateFromSandbox() error = %v", err)
	}
	if tpl.TemplateID != "python-ready" {
		t.Fatalf("TemplateID = %q, want python-ready", tpl.TemplateID)
	}
}

func TestCreateTemplateFromSandboxReturnsConflict(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusConflict, map[string]any{
			"success": false,
			"error": map[string]any{
				"code":    "conflict",
				"message": "template already exists",
			},
		})
	})
	defer server.Close()

	_, err := client.CreateTemplateFromSandbox(
		context.Background(),
		NewTemplateFromSandboxCreateRequest("python-ready", "sb_source", nil),
		nil,
	)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T %v, want *APIError", err, err)
	}
	if apiErr.StatusCode != http.StatusConflict || apiErr.Code != "conflict" {
		t.Fatalf("APIError = %+v", apiErr)
	}
}

func TestUpdateTemplateReturnsNotReadyConflict(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "3")
		writeJSON(t, w, http.StatusConflict, map[string]any{
			"success": false,
			"error": map[string]any{
				"code":    "template_not_ready",
				"message": "template creation is still in progress",
			},
		})
	})
	defer server.Close()

	_, err := client.UpdateTemplate(
		context.Background(),
		"python-ready",
		apispec.TemplateUpdateRequest{Spec: apispec.SandboxTemplateSpec{}},
	)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T %v, want *APIError", err, err)
	}
	if apiErr.StatusCode != http.StatusConflict ||
		apiErr.Code != "template_not_ready" ||
		apiErr.RetryAfterSeconds != 3 {
		t.Fatalf("APIError = %+v", apiErr)
	}
}

func TestWaitTemplateReady(t *testing.T) {
	var calls atomic.Int32
	client, server := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			writeTemplateResponse(t, w, http.StatusOK, "python-ready", "creating", "publishing")
			return
		}
		writeTemplateResponse(t, w, http.StatusOK, "python-ready", "ready", "reconciling")
	})
	defer server.Close()

	tpl, err := client.WaitTemplateReady(context.Background(), "python-ready", &WaitTemplateReadyOptions{
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("WaitTemplateReady() error = %v", err)
	}
	if tpl.TemplateID != "python-ready" || calls.Load() != 2 {
		t.Fatalf("template = %+v, calls = %d", tpl, calls.Load())
	}
}

func TestWaitTemplateReadyReturnsCreationFailure(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		now := time.Now().UTC()
		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"template_id": "python-ready",
				"scope":       "team",
				"spec":        map[string]any{},
				"status": map[string]any{
					"creation": map[string]any{
						"state":   "failed",
						"stage":   "publishing",
						"reason":  "RegistryPushFailed",
						"message": "registry unavailable",
					},
				},
				"created_at": now,
				"updated_at": now,
			},
		})
	})
	defer server.Close()

	_, err := client.WaitTemplateReady(context.Background(), "python-ready", nil)
	var failed *TemplateCreationFailedError
	if !errors.As(err, &failed) {
		t.Fatalf("error = %T %v, want *TemplateCreationFailedError", err, err)
	}
	if failed.Stage != apispec.TemplateCreationStatusStagePublishing || failed.Reason != "RegistryPushFailed" {
		t.Fatalf("failure = %+v", failed)
	}
}

func TestWaitTemplateReadyTreatsLegacyTemplateAsReady(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		now := time.Now().UTC()
		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"template_id": "legacy",
				"scope":       "team",
				"spec":        map[string]any{},
				"created_at":  now,
				"updated_at":  now,
			},
		})
	})
	defer server.Close()

	tpl, err := client.WaitTemplateReady(context.Background(), "legacy", nil)
	if err != nil {
		t.Fatalf("WaitTemplateReady() error = %v", err)
	}
	if tpl.TemplateID != "legacy" {
		t.Fatalf("TemplateID = %q, want legacy", tpl.TemplateID)
	}
}

func TestWaitTemplateReadyHonorsContext(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		writeTemplateResponse(t, w, http.StatusOK, "python-ready", "creating", "capturing")
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := client.WaitTemplateReady(ctx, "python-ready", &WaitTemplateReadyOptions{
		PollInterval: time.Second,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
}

func writeTemplateResponse(t *testing.T, w http.ResponseWriter, status int, templateID, state, stage string) {
	t.Helper()
	now := time.Now().UTC()
	writeJSON(t, w, status, map[string]any{
		"success": true,
		"data": map[string]any{
			"template_id": templateID,
			"scope":       "team",
			"spec":        map[string]any{},
			"status": map[string]any{
				"creation": map[string]any{
					"state": state,
					"stage": stage,
				},
			},
			"created_at": now,
			"updated_at": now,
		},
	})
}
