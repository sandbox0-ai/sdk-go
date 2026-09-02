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

func TestClaimSandboxWithOptions(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/sandboxes" {
			t.Fatalf("path = %s, want /api/v1/sandboxes", r.URL.Path)
		}

		body := decodeClaimRequest(t, r)

		if body.Template != "default" {
			t.Fatalf("template = %q, want default", body.Template)
		}
		config, ok := body.Config.Get()
		if !ok {
			t.Fatal("config not set")
		}
		ttl, ok := config.TTL.Get()
		if !ok || ttl != 300 {
			t.Fatalf("ttl = %d, want 300", ttl)
		}
		resources, ok := config.Resources.Get()
		if !ok {
			t.Fatal("resources not set")
		}
		memory, ok := resources.Memory.Get()
		if !ok || memory != "512Mi" {
			t.Fatalf("memory = %q, want 512Mi", memory)
		}
		snapshotID, ok := body.SnapshotID.Get()
		if !ok || snapshotID != "snap_123" {
			t.Fatalf("snapshot_id = %q, want snap_123", snapshotID)
		}

		writeJSON(t, w, http.StatusCreated, map[string]any{
			"success": true,
			"data": map[string]any{
				"sandbox_id": "sb_123",
				"template":   "default",
				"cluster_id": "cluster-a",
				"runtime_id": "alloc-a",
				"status":     "running",
			},
		})
	})
	defer server.Close()

	sandbox, err := client.ClaimSandbox(
		context.Background(),
		"default",
		WithSandboxTTL(300),
		WithSandboxMemory("512Mi"),
		WithSandboxSnapshotID("snap_123"),
	)
	if err != nil {
		t.Fatalf("ClaimSandbox() error = %v", err)
	}
	if sandbox == nil {
		t.Fatal("ClaimSandbox() returned nil sandbox")
	}
	if sandbox.ID != "sb_123" {
		t.Fatalf("sandbox.ID = %q, want sb_123", sandbox.ID)
	}
	if sandbox.ClusterID == nil || *sandbox.ClusterID != "cluster-a" {
		t.Fatalf("sandbox.ClusterID = %v, want cluster-a", sandbox.ClusterID)
	}
	if sandbox.RuntimeID != "alloc-a" {
		t.Fatalf("sandbox.RuntimeID = %q, want alloc-a", sandbox.RuntimeID)
	}
}

func TestClaimSandboxWithServicesOption(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body := decodeClaimRequest(t, r)

		config, ok := body.Config.Get()
		if !ok {
			t.Fatal("config not set")
		}
		if len(config.Services) != 1 {
			t.Fatalf("services count = %d, want 1", len(config.Services))
		}
		service := config.Services[0]
		if service.ID != "api" || service.Port.Or(0) != 8080 {
			t.Fatalf("service = %+v, want api service on port 8080", service)
		}
		if !service.Ingress.Public || len(service.Ingress.Routes) != 1 {
			t.Fatalf("ingress = %+v, want one public route", service.Ingress)
		}

		writeJSON(t, w, http.StatusCreated, map[string]any{
			"success": true,
			"data": map[string]any{
				"sandbox_id": "sb_123",
				"template":   "default",
				"runtime_id": "alloc-a",
				"status":     "running",
			},
		})
	})
	defer server.Close()

	_, err := client.ClaimSandbox(
		context.Background(),
		"default",
		WithSandboxServices([]apispec.SandboxAppService{
			{
				ID:   "api",
				Port: apispec.NewOptInt32(8080),
				Ingress: apispec.SandboxAppServiceIngress{
					Public: true,
					Routes: []apispec.SandboxAppServiceRoute{
						{ID: "api", Resume: true},
					},
				},
			},
		}),
	)
	if err != nil {
		t.Fatalf("ClaimSandbox() error = %v", err)
	}
}

func TestClaimSandboxReturnsClaimStartThrottledAPIError(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		writeJSON(t, w, http.StatusTooManyRequests, map[string]any{
			"success": false,
			"error": map[string]any{
				"code":    CodeClaimStartThrottled,
				"message": "claim start admission throttled",
			},
		})
	})
	defer server.Close()

	_, err := client.ClaimSandbox(context.Background(), "default")
	if err == nil {
		t.Fatal("ClaimSandbox() error = nil, want throttled error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusTooManyRequests)
	}
	if apiErr.Code != CodeClaimStartThrottled {
		t.Fatalf("Code = %q, want %q", apiErr.Code, CodeClaimStartThrottled)
	}
	if apiErr.RetryAfterSeconds != 2 {
		t.Fatalf("RetryAfterSeconds = %d, want 2", apiErr.RetryAfterSeconds)
	}
	if !apiErr.IsClaimStartThrottled() {
		t.Fatal("IsClaimStartThrottled() = false, want true")
	}
	if !IsClaimStartThrottled(err) {
		t.Fatal("IsClaimStartThrottled(err) = false, want true")
	}
}

func TestUpdateSandboxLifecycleBuildsRequest(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/api/v1/sandboxes/sb_123" {
			t.Fatalf("path = %s, want /api/v1/sandboxes/sb_123", r.URL.Path)
		}

		var body apispec.SandboxUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode update request: %v", err)
		}
		config, ok := body.Config.Get()
		if !ok {
			t.Fatal("config not set")
		}
		ttl, ok := config.TTL.Get()
		if !ok || ttl != 600 {
			t.Fatalf("ttl = %d, want 600", ttl)
		}
		autoResume, ok := config.AutoResume.Get()
		if !ok || !autoResume {
			t.Fatalf("auto_resume = %v, %v; want true, true", autoResume, ok)
		}

		payload := sandboxJSON("sb_123")
		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data":    payload,
		})
	})
	defer server.Close()

	sandbox, err := client.UpdateSandbox(context.Background(), "sb_123", apispec.SandboxUpdateRequest{
		Config: apispec.NewOptSandboxUpdateConfig(apispec.SandboxUpdateConfig{
			TTL:        apispec.NewOptInt32(600),
			AutoResume: apispec.NewOptBool(true),
		}),
	})
	if err != nil {
		t.Fatalf("UpdateSandbox() error = %v", err)
	}
	if sandbox.ID != "sb_123" {
		t.Fatalf("sandbox.ID = %q, want sb_123", sandbox.ID)
	}
}

func TestPauseSandboxAndWaitPollsCommittedProjection(t *testing.T) {
	var getCalls atomic.Int32
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/sandboxes/sb_123/pause":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"sandbox_id": "sb_123",
					"paused":     true,
					"status":     "paused",
				},
			})
		case "GET /api/v1/sandboxes/sb_123":
			payload := sandboxJSON("sb_123")
			if getCalls.Add(1) == 1 {
				payload["status"] = "running"
				payload["paused"] = false
				payload["runtime_id"] = "alloc-a"
			}
			writeJSON(t, w, http.StatusOK, map[string]any{"success": true, "data": payload})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	sandbox, err := client.PauseSandboxAndWait(context.Background(), "sb_123", &SandboxLifecycleWaitOptions{
		Timeout:      time.Second,
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("PauseSandboxAndWait() error = %v", err)
	}
	if sandbox.Status != apispec.SandboxLifecycleStatusPaused || !sandbox.Paused {
		t.Fatalf("sandbox = %+v, want committed paused projection", sandbox)
	}
	if getCalls.Load() < 2 {
		t.Fatalf("GET calls = %d, want at least 2", getCalls.Load())
	}
}

func TestResumeSandboxAndWaitRequiresNextRuntimeGeneration(t *testing.T) {
	var getCalls atomic.Int32
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/sandboxes/sb_123/resume":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"sandbox_id": "sb_123",
					"resumed":    true,
				},
			})
		case "GET /api/v1/sandboxes/sb_123":
			payload := sandboxJSON("sb_123")
			if getCalls.Add(1) >= 3 {
				payload["status"] = "running"
				payload["paused"] = false
				payload["runtime_id"] = "alloc-b"
				payload["runtime_generation"] = 2
			}
			writeJSON(t, w, http.StatusOK, map[string]any{"success": true, "data": payload})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	sandbox, err := client.ResumeSandboxAndWait(context.Background(), "sb_123", &SandboxLifecycleWaitOptions{
		Timeout:      time.Second,
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("ResumeSandboxAndWait() error = %v", err)
	}
	if sandbox.Status != apispec.SandboxLifecycleStatusRunning || sandbox.RuntimeGeneration != 2 {
		t.Fatalf("sandbox = %+v, want running generation 2", sandbox)
	}
}

func TestWaitForSandboxLifecycleTimesOutWithLastProjection(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data":    sandboxJSON("sb_123"),
		})
	})
	defer server.Close()

	_, err := client.WaitForSandboxLifecycle(
		context.Background(),
		"sb_123",
		func(*apispec.Sandbox) bool { return false },
		&SandboxLifecycleWaitOptions{Timeout: 10 * time.Millisecond, PollInterval: time.Millisecond},
	)
	var timeoutErr *SandboxWaitTimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Fatalf("error = %v, want *SandboxWaitTimeoutError", err)
	}
	if timeoutErr.LastSandbox == nil || timeoutErr.LastSandbox.ID != "sb_123" {
		t.Fatalf("last sandbox = %+v, want sb_123", timeoutErr.LastSandbox)
	}
}

func TestSandboxRootFSOperationsUseGeneratedAPI(t *testing.T) {
	calls := make([]string, 0, 7)
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		calls = append(calls, key)

		switch key {
		case "POST /api/v1/sandboxes/sb_1/snapshots":
			var req apispec.CreateSandboxRootFSSnapshotRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode create snapshot request: %v", err)
			}
			name, ok := req.Name.Get()
			if !ok || name != "snap" {
				t.Fatalf("snapshot name = %q, %v; want snap, true", name, ok)
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data":    sandboxRootFSSnapshotJSON("snap_1", "sb_1"),
			})
		case "GET /api/v1/sandboxes/sb_1/snapshots":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"snapshots": []map[string]any{sandboxRootFSSnapshotJSON("snap_1", "sb_1")},
					"count":     1,
				},
			})
		case "GET /api/v1/sandbox-rootfs-snapshots/snap_1":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data":    sandboxRootFSSnapshotJSON("snap_1", "sb_1"),
			})
		case "DELETE /api/v1/sandbox-rootfs-snapshots/snap_1":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data":    map[string]any{"deleted": true},
			})
		case "POST /api/v1/sandboxes/sb_1/rootfs/restore":
			var req apispec.RestoreSandboxRootFSRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode restore request: %v", err)
			}
			if req.SnapshotID != "snap_1" {
				t.Fatalf("snapshot_id = %q, want snap_1", req.SnapshotID)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"sandbox_id":  "sb_1",
					"snapshot_id": "snap_1",
					"status":      "paused",
				},
			})
		case "PUT /api/v1/sandboxes/sb_1/rootfs/rebase":
			var req apispec.RebaseSandboxRootFSRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode rebase request: %v", err)
			}
			if req.TargetBaseArtifactDigest != "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
				t.Fatalf("target_base_artifact_digest = %q", req.TargetBaseArtifactDigest)
			}
			if ttl, ok := req.RollbackTTL.Get(); !ok || ttl != 3600 {
				t.Fatalf("rollback_ttl = %d, %v; want 3600, true", ttl, ok)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"sandbox_id":           "sb_1",
					"generation_id":        "gen_2",
					"base_artifact_digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"rollback_expires_at":  "2026-01-02T04:04:05Z",
					"status":               "paused",
				},
			})
		case "POST /api/v1/sandboxes/sb_1/fork":
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode fork request: %v", err)
			}
			if len(req) != 0 {
				t.Fatalf("fork request = %+v, want empty object", req)
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"source_sandbox_id": "sb_1",
					"sandbox":           sandboxJSON("sb_2"),
				},
			})
		default:
			t.Fatalf("unexpected request %s", key)
		}
	})
	defer server.Close()

	ctx := context.Background()
	snapshot, err := client.CreateSandboxRootFSSnapshot(ctx, "sb_1", &apispec.CreateSandboxRootFSSnapshotRequest{
		Name: apispec.NewOptString("snap"),
	})
	if err != nil {
		t.Fatalf("CreateSandboxRootFSSnapshot() error = %v", err)
	}
	if snapshot.ID != "snap_1" {
		t.Fatalf("snapshot.ID = %q, want snap_1", snapshot.ID)
	}

	snapshots, err := client.ListSandboxRootFSSnapshots(ctx, "sb_1")
	if err != nil {
		t.Fatalf("ListSandboxRootFSSnapshots() error = %v", err)
	}
	if snapshots.Count != 1 || len(snapshots.Snapshots) != 1 {
		t.Fatalf("snapshots = %+v, want one snapshot", snapshots)
	}

	fetched, err := client.GetSandboxRootFSSnapshot(ctx, "snap_1")
	if err != nil {
		t.Fatalf("GetSandboxRootFSSnapshot() error = %v", err)
	}
	if fetched.SandboxID != "sb_1" {
		t.Fatalf("fetched.SandboxID = %q, want sb_1", fetched.SandboxID)
	}

	deleted, err := client.DeleteSandboxRootFSSnapshot(ctx, "snap_1")
	if err != nil {
		t.Fatalf("DeleteSandboxRootFSSnapshot() error = %v", err)
	}
	if !deleted.Success {
		t.Fatal("deleted.Success = false, want true")
	}

	restored, err := client.RestoreSandboxRootFS(ctx, "sb_1", apispec.RestoreSandboxRootFSRequest{SnapshotID: "snap_1"})
	if err != nil {
		t.Fatalf("RestoreSandboxRootFS() error = %v", err)
	}
	if restored.SnapshotID != "snap_1" || restored.Status != apispec.SandboxLifecycleStatusPaused {
		t.Fatalf("restored = %+v, want snapshot snap_1 paused", restored)
	}

	rebased, err := client.RebaseSandboxRootFS(ctx, "sb_1", apispec.RebaseSandboxRootFSRequest{
		TargetBaseArtifactDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		RollbackTTL:              apispec.NewOptInt32(3600),
	})
	if err != nil {
		t.Fatalf("RebaseSandboxRootFS() error = %v", err)
	}
	if rebased.GenerationID != "gen_2" || rebased.Status != apispec.SandboxLifecycleStatusPaused {
		t.Fatalf("rebased = %+v, want generation gen_2 paused", rebased)
	}

	forked, err := client.ForkSandbox(ctx, "sb_1", nil)
	if err != nil {
		t.Fatalf("ForkSandbox() error = %v", err)
	}
	if forked.SourceSandboxID != "sb_1" || forked.Sandbox.ID != "sb_2" {
		t.Fatalf("forked = %+v, want source sb_1 and sandbox sb_2", forked)
	}

	wantCalls := []string{
		"POST /api/v1/sandboxes/sb_1/snapshots",
		"GET /api/v1/sandboxes/sb_1/snapshots",
		"GET /api/v1/sandbox-rootfs-snapshots/snap_1",
		"DELETE /api/v1/sandbox-rootfs-snapshots/snap_1",
		"POST /api/v1/sandboxes/sb_1/rootfs/restore",
		"PUT /api/v1/sandboxes/sb_1/rootfs/rebase",
		"POST /api/v1/sandboxes/sb_1/fork",
	}
	if len(calls) != len(wantCalls) {
		t.Fatalf("calls = %v, want %v", calls, wantCalls)
	}
	for i := range wantCalls {
		if calls[i] != wantCalls[i] {
			t.Fatalf("calls[%d] = %q, want %q", i, calls[i], wantCalls[i])
		}
	}
}

func TestForkSandboxBuildsLifecycleOverrideRequest(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/sandboxes/sb_source/fork" {
			t.Fatalf("path = %s, want /api/v1/sandboxes/sb_source/fork", r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got != "fork-request-one" {
			t.Fatalf("Idempotency-Key = %q, want fork-request-one", got)
		}

		var req apispec.ForkSandboxRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode fork request: %v", err)
		}
		config, ok := req.Config.Get()
		if !ok {
			t.Fatal("config not set")
		}
		ttl, ok := config.TTL.Get()
		if !ok || ttl != 60 {
			t.Fatalf("ttl = %d, want 60", ttl)
		}
		hardTTL, ok := config.HardTTL.Get()
		if !ok || hardTTL != 120 {
			t.Fatalf("hard_ttl = %d, want 120", hardTTL)
		}

		writeJSON(t, w, http.StatusCreated, map[string]any{
			"success": true,
			"data": map[string]any{
				"source_sandbox_id": "sb_source",
				"sandbox":           sandboxJSON("sb_fork"),
			},
		})
	})
	defer server.Close()

	forked, err := client.ForkSandboxWithOptions(context.Background(), "sb_source", &apispec.ForkSandboxRequest{
		Config: apispec.NewOptForkSandboxConfig(apispec.ForkSandboxConfig{
			TTL:     apispec.NewOptInt32(60),
			HardTTL: apispec.NewOptInt32(120),
		}),
	}, &ForkSandboxOptions{IdempotencyKey: "fork-request-one"})
	if err != nil {
		t.Fatalf("ForkSandbox() error = %v", err)
	}
	if forked.SourceSandboxID != "sb_source" || forked.Sandbox.ID != "sb_fork" {
		t.Fatalf("forked = %+v, want source sb_source and sandbox sb_fork", forked)
	}
}

func decodeClaimRequest(t *testing.T, r *http.Request) apispec.ClaimRequest {
	t.Helper()

	defer func() {
		_ = r.Body.Close()
	}()

	var req apispec.ClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatalf("decode claim request: %v", err)
	}
	return req
}

func sandboxRootFSSnapshotJSON(snapshotID, sandboxID string) map[string]any {
	return map[string]any{
		"id":          snapshotID,
		"sandbox_id":  sandboxID,
		"name":        "snap",
		"description": "test snapshot",
		"created_at":  "2026-01-02T03:04:05Z",
	}
}

func sandboxJSON(id string) map[string]any {
	return map[string]any{
		"id":                 id,
		"template_id":        "default",
		"team_id":            "team_1",
		"status":             "paused",
		"paused":             true,
		"auto_resume":        false,
		"services":           []map[string]any{},
		"runtime_id":         "",
		"runtime_generation": 1,
		"expires_at":         "2026-01-02T04:04:05Z",
		"hard_expires_at":    "2026-01-03T03:04:05Z",
		"claimed_at":         "2026-01-02T03:04:05Z",
		"created_at":         "2026-01-02T03:04:05Z",
		"updated_at":         "2026-01-02T03:04:05Z",
	}
}
