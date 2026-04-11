package sandbox0

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestStartImageBuild(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/image-builds" {
			t.Fatalf("path = %s, want /api/v1/image-builds", r.URL.Path)
		}
		body := decodeImageBuildRequest(t, r)
		if body.ContextVolumeID != "sv-context" {
			t.Fatalf("context_volume_id = %q, want sv-context", body.ContextVolumeID)
		}
		if got, ok := body.ContextPath.Get(); !ok || got != "src" {
			t.Fatalf("context_path = %q, %v, want src", got, ok)
		}
		if got, ok := body.DockerfilePath.Get(); !ok || got != "docker/Dockerfile" {
			t.Fatalf("dockerfile_path = %q, %v, want docker/Dockerfile", got, ok)
		}
		if got, ok := body.CacheVolumeID.Get(); !ok || got != "sv-cache" {
			t.Fatalf("cache_volume_id = %q, %v, want sv-cache", got, ok)
		}
		if got, ok := body.Platform.Get(); !ok || got != "linux/amd64" {
			t.Fatalf("platform = %q, %v, want linux/amd64", got, ok)
		}
		if got, ok := body.NoCache.Get(); !ok || !got {
			t.Fatalf("no_cache = %v, %v, want true", got, ok)
		}
		if got, ok := body.Pull.Get(); !ok || !got {
			t.Fatalf("pull = %v, %v, want true", got, ok)
		}
		args, ok := body.BuildArgs.Get()
		if !ok || args["A"] != "1" {
			t.Fatalf("build_args = %#v, %v, want A=1", args, ok)
		}

		writeJSON(t, w, http.StatusCreated, map[string]any{
			"success": true,
			"data": map[string]any{
				"sandbox_id":          "sb-build",
				"context_id":          "ctx-build",
				"status":              "running",
				"builder_template":    "sandbox0-image-builder",
				"target_image":        "image-builds/build-123:latest",
				"push_image":          "registry.local/team/image-builds/build-123:latest",
				"pull_image":          "registry.pull/team/image-builds/build-123:latest",
				"sandbox_path":        "/api/v1/sandboxes/sb-build",
				"context_api_path":    "/api/v1/sandboxes/sb-build/contexts/ctx-build",
				"context_stream_path": "/api/v1/sandboxes/sb-build/contexts/ctx-build/ws",
			},
		})
	})
	defer server.Close()

	build, err := client.StartImageBuild(context.Background(), "sv-context",
		WithImageBuildContextPath("src"),
		WithImageBuildDockerfilePath("docker/Dockerfile"),
		WithImageBuildCacheVolume("sv-cache"),
		WithImageBuildPlatform("linux/amd64"),
		WithImageBuildNoCache(),
		WithImageBuildPull(),
		WithImageBuildArgs(map[string]string{"A": "1"}),
	)
	if err != nil {
		t.Fatalf("StartImageBuild() error = %v", err)
	}
	if build.TargetImage != "image-builds/build-123:latest" {
		t.Fatalf("target image = %q, want image-builds/build-123:latest", build.TargetImage)
	}
	if build.Status != apispec.ImageBuildResponseStatusRunning {
		t.Fatalf("status = %q, want running", build.Status)
	}
}

func TestStartImageBuildReturnsAPIError(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusServiceUnavailable, map[string]any{
			"success": false,
			"error": map[string]any{
				"code":    "unavailable",
				"message": "image build is not configured",
			},
		})
	})
	defer server.Close()

	_, err := client.StartImageBuild(context.Background(), "sv-context")
	if err == nil {
		t.Fatal("StartImageBuild() error = nil, want error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusServiceUnavailable || apiErr.Code != "unavailable" {
		t.Fatalf("api error = %#v, want 503 unavailable", apiErr)
	}
}

func decodeImageBuildRequest(t *testing.T, r *http.Request) apispec.ImageBuildRequest {
	t.Helper()
	defer r.Body.Close()
	var req apispec.ImageBuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatalf("decode image build request: %v", err)
	}
	return req
}
