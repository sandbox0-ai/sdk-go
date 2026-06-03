package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestFilesystemClientLifecycle(t *testing.T) {
	seen := map[string]bool{}
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		seen[r.Method+" "+r.URL.Path] = true
		switch r.Method + " " + r.URL.Path {
		case "POST /api/v1/sandboxfilesystems":
			var req apispec.CreateSandboxFilesystemRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode create filesystem request: %v", err)
			}
			template, ok := req.Template.Get()
			if !ok || template != "default" {
				t.Fatalf("template = %q, want default", template)
			}
			writeJSON(t, w, http.StatusCreated, filesystemResponseBody("fs_1"))
		case "POST /api/v1/sandboxfilesystems/fs_1/fork":
			writeJSON(t, w, http.StatusCreated, filesystemResponseBody("fs_2"))
		case "POST /api/v1/sandboxfilesystems/fs_2/snapshots":
			var req apispec.CreateSandboxFilesystemSnapshotRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode snapshot request: %v", err)
			}
			if req.Name != "snap-a" {
				t.Fatalf("snapshot name = %q, want snap-a", req.Name)
			}
			writeJSON(t, w, http.StatusCreated, filesystemSnapshotResponseBody("snap_1"))
		case "POST /api/v1/sandboxfilesystems/fs_2/snapshots/snap_1/restore":
			writeJSON(t, w, http.StatusOK, filesystemResponseBody("fs_2"))
		default:
			http.NotFound(w, r)
		}
	})
	defer server.Close()

	filesystem, err := client.CreateFilesystem(context.Background(), apispec.CreateSandboxFilesystemRequest{
		Template:        apispec.NewOptString("default"),
		BaseImageDigest: apispec.NewOptString("sha256:base"),
	})
	if err != nil {
		t.Fatalf("CreateFilesystem() error = %v", err)
	}
	if filesystem.ID != "fs_1" {
		t.Fatalf("filesystem id = %q, want fs_1", filesystem.ID)
	}
	forked, err := client.ForkFilesystem(context.Background(), "fs_1", nil)
	if err != nil {
		t.Fatalf("ForkFilesystem() error = %v", err)
	}
	if forked.ID != "fs_2" {
		t.Fatalf("forked id = %q, want fs_2", forked.ID)
	}
	snapshot, err := client.CreateFilesystemSnapshot(context.Background(), "fs_2", apispec.CreateSandboxFilesystemSnapshotRequest{Name: "snap-a"})
	if err != nil {
		t.Fatalf("CreateFilesystemSnapshot() error = %v", err)
	}
	if snapshot.ID != "snap_1" {
		t.Fatalf("snapshot id = %q, want snap_1", snapshot.ID)
	}
	restored, err := client.RestoreFilesystemSnapshot(context.Background(), "fs_2", "snap_1")
	if err != nil {
		t.Fatalf("RestoreFilesystemSnapshot() error = %v", err)
	}
	if restored.ID != "fs_2" {
		t.Fatalf("restored id = %q, want fs_2", restored.ID)
	}

	for _, key := range []string{
		"POST /api/v1/sandboxfilesystems",
		"POST /api/v1/sandboxfilesystems/fs_1/fork",
		"POST /api/v1/sandboxfilesystems/fs_2/snapshots",
		"POST /api/v1/sandboxfilesystems/fs_2/snapshots/snap_1/restore",
	} {
		if !seen[key] {
			t.Fatalf("missing request %s; seen=%#v", key, seen)
		}
	}
}

func filesystemResponseBody(id string) map[string]any {
	return map[string]any{
		"success": true,
		"data": map[string]any{
			"id":                id,
			"team_id":           "team_1",
			"user_id":           "user_1",
			"base_image_digest": "sha256:base",
			"s0fs_head":         "manifests/00000000000000000001.json",
			"state":             "available",
			"created_at":        "2026-01-01T00:00:00Z",
			"updated_at":        "2026-01-01T00:00:00Z",
		},
	}
}

func filesystemSnapshotResponseBody(id string) map[string]any {
	return map[string]any{
		"success": true,
		"data": map[string]any{
			"id":                id,
			"filesystem_id":     "fs_2",
			"team_id":           "team_1",
			"user_id":           "user_1",
			"base_image_digest": "sha256:base",
			"s0fs_head":         "manifests/00000000000000000001.json",
			"name":              "snap-a",
			"size_bytes":        1,
			"created_at":        "2026-01-01T00:00:00Z",
		},
	}
}
