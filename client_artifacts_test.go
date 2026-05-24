package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestClientArtifacts(t *testing.T) {
	t.Run("create from volume", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/api/v1/artifacts" {
				t.Fatalf("path = %s, want /api/v1/artifacts", r.URL.Path)
			}
			var body apispec.CreateArtifactRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create request: %v", err)
			}
			if body.Source.Type != apispec.CreateArtifactSourceTypeSandboxVolume || body.Source.SandboxvolumeID != "vol-1" {
				t.Fatalf("source = %#v, want volume vol-1", body.Source)
			}
			if got := body.Name.Or(""); got != "bundle" {
				t.Fatalf("name = %q, want bundle", got)
			}
			if got := body.Kind.Or(""); got != "nextjs" {
				t.Fatalf("kind = %q, want nextjs", got)
			}
			if got := body.MediaType.Or(""); got != "application/vnd.sandbox0.volume" {
				t.Fatalf("media type = %q, want application/vnd.sandbox0.volume", got)
			}
			writeJSON(t, w, http.StatusCreated, artifactResponse())
		})
		defer server.Close()

		artifact, err := client.CreateArtifactFromVolume(
			context.Background(),
			"vol-1",
			WithArtifactName("bundle"),
			WithArtifactKind("nextjs"),
			WithArtifactMediaType("application/vnd.sandbox0.volume"),
		)
		if err != nil {
			t.Fatalf("CreateArtifactFromVolume() error = %v", err)
		}
		if artifact.ID != "art-1" || artifact.SourceVolumeID != "vol-1" {
			t.Fatalf("artifact = %#v, want art-1 from vol-1", artifact)
		}
	})

	t.Run("list get delete and materialize", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/artifacts":
				writeJSON(t, w, http.StatusOK, map[string]any{
					"success": true,
					"data": map[string]any{
						"artifacts": []any{artifactMap()},
					},
				})
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/artifacts/art-1":
				writeJSON(t, w, http.StatusOK, artifactResponse())
			case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/artifacts/art-1":
				writeJSON(t, w, http.StatusOK, map[string]any{
					"success": true,
					"data": map[string]any{
						"deleted": true,
					},
				})
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/artifacts/art-1/volume":
				var body apispec.CreateArtifactVolumeRequest
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode materialize request: %v", err)
				}
				if got := body.AccessMode.Or(""); got != apispec.VolumeAccessModeROX {
					t.Fatalf("access mode = %q, want ROX", got)
				}
				writeJSON(t, w, http.StatusCreated, map[string]any{
					"success": true,
					"data":    volumeMap("vol-rox", "ROX"),
				})
			default:
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
			}
		})
		defer server.Close()

		artifacts, err := client.ListArtifacts(context.Background())
		if err != nil {
			t.Fatalf("ListArtifacts() error = %v", err)
		}
		if len(artifacts) != 1 || artifacts[0].ID != "art-1" {
			t.Fatalf("artifacts = %#v, want art-1", artifacts)
		}

		artifact, err := client.GetArtifact(context.Background(), "art-1")
		if err != nil {
			t.Fatalf("GetArtifact() error = %v", err)
		}
		if artifact.Name != "bundle" {
			t.Fatalf("artifact name = %q, want bundle", artifact.Name)
		}

		volume, err := client.MaterializeArtifactVolume(context.Background(), "art-1", &apispec.CreateArtifactVolumeRequest{
			AccessMode: apispec.NewOptVolumeAccessMode(apispec.VolumeAccessModeROX),
		})
		if err != nil {
			t.Fatalf("MaterializeArtifactVolume() error = %v", err)
		}
		if volume.ID != "vol-rox" {
			t.Fatalf("volume id = %q, want vol-rox", volume.ID)
		}

		if _, err := client.DeleteArtifact(context.Background(), "art-1"); err != nil {
			t.Fatalf("DeleteArtifact() error = %v", err)
		}
	})
}

func artifactResponse() map[string]any {
	return map[string]any{
		"success": true,
		"data":    artifactMap(),
	}
}

func artifactMap() map[string]any {
	return map[string]any{
		"id":               "art-1",
		"team_id":          "team-1",
		"user_id":          "user-1",
		"name":             "bundle",
		"kind":             "nextjs",
		"media_type":       "application/vnd.sandbox0.volume",
		"digest":           "sha256:abc",
		"source_volume_id": "vol-1",
		"snapshot_id":      "snap-1",
		"size_bytes":       42,
		"metadata":         map[string]any{},
		"created_at":       "2026-05-24T00:00:00Z",
		"updated_at":       "2026-05-24T00:00:00Z",
	}
}

func volumeMap(id, accessMode string) map[string]any {
	return map[string]any{
		"id":                id,
		"team_id":           "team-1",
		"user_id":           "user-1",
		"source_volume_id":  nil,
		"default_posix_uid": 0,
		"default_posix_gid": 0,
		"access_mode":       accessMode,
		"created_at":        "2026-05-24T00:00:00Z",
		"updated_at":        "2026-05-24T00:00:00Z",
	}
}
