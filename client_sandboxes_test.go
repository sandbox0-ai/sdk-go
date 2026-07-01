package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestClaimSandboxWithBootstrapMountOptions(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/sandboxes" {
			t.Fatalf("path = %s, want /api/v1/sandboxes", r.URL.Path)
		}

		body := decodeClaimRequest(t, r)

		template, ok := body.Template.Get()
		if !ok || template != "default" {
			t.Fatalf("template = %q, want default", template)
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
		if len(body.Mounts) != 2 {
			t.Fatalf("mount count = %d, want 2", len(body.Mounts))
		}
		if body.Mounts[0].SandboxvolumeID != "vol_1" || body.Mounts[0].MountPoint != "/workspace/data" {
			t.Fatalf("mount[0] = %+v, want vol_1:/workspace/data", body.Mounts[0])
		}
		if body.Mounts[1].SandboxvolumeID != "vol_2" || body.Mounts[1].MountPoint != "/workspace/readonly" {
			t.Fatalf("mount[1] = %+v, want vol_2:/workspace/readonly", body.Mounts[1])
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
				"pod_name":   "pod-a",
				"status":     "running",
				"bootstrap_mounts": []map[string]any{
					{
						"sandboxvolume_id": "vol_1",
						"mount_point":      "/workspace/data",
						"state":            "mounted",
					},
				},
			},
		})
	})
	defer server.Close()

	sandbox, err := client.ClaimSandbox(
		context.Background(),
		"default",
		WithSandboxTTL(300),
		WithSandboxMemory("512Mi"),
		WithSandboxBootstrapMount("vol_1", "/workspace/data"),
		WithSandboxBootstrapMounts(SandboxBootstrapMount{
			SandboxVolumeID: "vol_2",
			MountPoint:      "/workspace/readonly",
		}),
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
	if len(sandbox.BootstrapMounts) != 1 {
		t.Fatalf("bootstrap mount count = %d, want 1", len(sandbox.BootstrapMounts))
	}
	if sandbox.BootstrapMounts[0].SandboxvolumeID != "vol_1" {
		t.Fatalf("bootstrap mount volume id = %q, want vol_1", sandbox.BootstrapMounts[0].SandboxvolumeID)
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
				"pod_name":   "pod-a",
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

func TestUpdateSandboxMemoryBuildsRequest(t *testing.T) {
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
		resources, ok := config.Resources.Get()
		if !ok {
			t.Fatal("resources not set")
		}
		memory, ok := resources.Memory.Get()
		if !ok || memory != "2Gi" {
			t.Fatalf("memory = %q, want 2Gi", memory)
		}

		payload := sandboxJSON("sb_123")
		payload["resources"] = map[string]any{"memory": "2Gi"}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data":    payload,
		})
	})
	defer server.Close()

	sandbox, err := client.UpdateSandboxMemory(context.Background(), "sb_123", "2Gi")
	if err != nil {
		t.Fatalf("UpdateSandboxMemory() error = %v", err)
	}
	resources, ok := sandbox.Resources.Get()
	if !ok {
		t.Fatal("sandbox resources not set")
	}
	memory, ok := resources.Memory.Get()
	if !ok || memory != "2Gi" {
		t.Fatalf("sandbox memory = %q, want 2Gi", memory)
	}
}

func TestSandboxRootFSOperationsUseGeneratedAPI(t *testing.T) {
	calls := make([]string, 0, 6)
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
			writeJSON(t, w, http.StatusOK, map[string]any{"success": true})
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

	forked, err := client.ForkSandbox(context.Background(), "sb_source", &apispec.ForkSandboxRequest{
		Config: apispec.NewOptForkSandboxConfig(apispec.ForkSandboxConfig{
			TTL:     apispec.NewOptInt32(60),
			HardTTL: apispec.NewOptInt32(120),
		}),
	})
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
		"mounts":             []map[string]any{},
		"pod_name":           "",
		"runtime_generation": 1,
		"expires_at":         "2026-01-02T04:04:05Z",
		"hard_expires_at":    "2026-01-03T03:04:05Z",
		"claimed_at":         "2026-01-02T03:04:05Z",
		"created_at":         "2026-01-02T03:04:05Z",
		"updated_at":         "2026-01-02T03:04:05Z",
	}
}
