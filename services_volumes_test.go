package sandbox0

import (
	"context"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestVolumeServiceSuccess(t *testing.T) {
	volume := map[string]any{
		"id":          "vol-1",
		"buffer_size": "1Gi",
		"cache_size":  "2Gi",
		"created_at":  "2024-01-02T00:00:00Z",
		"updated_at":  "2024-01-02T00:00:00Z",
		"team_id":     "team-1",
		"user_id":     "user-1",
	}
	snapshot := map[string]any{
		"id":         "snap-1",
		"name":       "snap",
		"created_at": "2024-01-02T00:00:00Z",
		"expires_at": "2024-01-03T00:00:00Z",
		"size_bytes": 123,
		"volume_id":  "vol-1",
	}
	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxvolumes"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data":    volume,
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxvolumes"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data":    []map[string]any{volume},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxvolumes/vol-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data":    volume,
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxvolumes/vol-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"deleted": true,
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/sandboxvolumes/mount"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"mount_point":      "/mnt",
					"mounted_at":       "2024-01-02T00:00:00Z",
					"sandboxvolume_id": "vol-1",
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/sb-1/sandboxvolumes/unmount"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"unmounted": true,
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/sb-1/sandboxvolumes/status"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"mounts": []map[string]any{
						{"sandboxvolume_id": "vol-1"},
					},
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxvolumes/vol-1/snapshots"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data":    snapshot,
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxvolumes/vol-1/snapshots"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data":    []map[string]any{snapshot},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxvolumes/vol-1/snapshots/snap-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data":    snapshot,
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxvolumes/vol-1/snapshots/snap-1"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"deleted": true,
				},
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxvolumes/vol-1/snapshots/snap-1/restore"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"restored": true,
				},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	ctx := context.Background()

	if _, err := client.Volumes.Create(ctx, apispec.CreateSandboxVolumeRequest{}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if _, err := client.Volumes.List(ctx); err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if _, err := client.Volumes.Get(ctx, "vol-1"); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if _, err := client.Volumes.Delete(ctx, "vol-1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := client.Volumes.Mount(ctx, "sb-1", "vol-1", "/mnt", nil); err != nil {
		t.Fatalf("mount failed: %v", err)
	}
	if _, err := client.Volumes.Unmount(ctx, "sb-1", "vol-1"); err != nil {
		t.Fatalf("unmount failed: %v", err)
	}
	if _, err := client.Volumes.MountStatus(ctx, "sb-1"); err != nil {
		t.Fatalf("mount status failed: %v", err)
	}
	if _, err := client.Volumes.CreateSnapshot(ctx, "vol-1", apispec.CreateSnapshotRequest{Name: "snap"}); err != nil {
		t.Fatalf("create snapshot failed: %v", err)
	}
	if _, err := client.Volumes.ListSnapshots(ctx, "vol-1"); err != nil {
		t.Fatalf("list snapshots failed: %v", err)
	}
	if _, err := client.Volumes.GetSnapshot(ctx, "vol-1", "snap-1"); err != nil {
		t.Fatalf("get snapshot failed: %v", err)
	}
	if _, err := client.Volumes.DeleteSnapshot(ctx, "vol-1", "snap-1"); err != nil {
		t.Fatalf("delete snapshot failed: %v", err)
	}
	if _, err := client.Volumes.RestoreSnapshot(ctx, "vol-1", "snap-1"); err != nil {
		t.Fatalf("restore snapshot failed: %v", err)
	}
}

func TestVolumeServiceErrors(t *testing.T) {
	routes := routeMap{
		routeKey(http.MethodGet, "/api/v1/sandboxvolumes/vol-404"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusNotFound, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "not_found",
					"message": "missing",
				},
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxvolumes/vol-409"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusConflict, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "conflict",
					"message": "in use",
				},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxvolumes/vol-1/snapshots/snap-404"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusNotFound, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "not_found",
					"message": "missing",
				},
			})
		},
	}
	server := newTestServer(t, routes)
	defer server.Close()

	client := newTestClient(t, server)
	ctx := context.Background()

	if _, err := client.Volumes.Get(ctx, "vol-404"); err == nil {
		t.Fatal("expected not found error")
	}
	if _, err := client.Volumes.Delete(ctx, "vol-409"); err == nil {
		t.Fatal("expected conflict error")
	}
	if _, err := client.Volumes.GetSnapshot(ctx, "vol-1", "snap-404"); err == nil {
		t.Fatal("expected not found snapshot error")
	}
}
