package sandbox0

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

type routeMap map[string]http.HandlerFunc

func routeKey(method, path string) string {
	return method + " " + path
}

func newTestServer(t *testing.T, routes routeMap) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler, ok := routes[routeKey(r.Method, r.URL.Path)]; ok {
			handler(w, r)
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
}

func newTestClient(t *testing.T, server *httptest.Server, opts ...Option) *Client {
	t.Helper()
	allOpts := append([]Option{WithBaseURL(server.URL), WithToken("test-token")}, opts...)
	client, err := NewClient(allOpts...)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write JSON response: %v", err)
	}
}

func newClaimResponse(id string) apispec.ClaimResponse {
	return apispec.ClaimResponse{
		SandboxID: id,
		Status:    "running",
		PodName:   "sandbox-" + id,
		Template:  "default",
	}
}

func newSandbox(id string) apispec.Sandbox {
	now := time.Now().UTC()
	return apispec.Sandbox{
		ID:         id,
		TemplateID: "default",
		TeamID:     "team-1",
		UserID:     apispec.NewOptString("user-1"),
		Status:     "running",
		PodName:    "sandbox-" + id,
		ExpiresAt:  now.Add(10 * time.Minute),
		ClaimedAt:  now,
		CreatedAt:  now,
	}
}

func newTemplate(id string) apispec.Template {
	now := time.Now().UTC()
	return apispec.Template{
		TemplateID: id,
		Scope:      "team",
		TeamID:     apispec.NewOptString("team-1"),
		UserID:     apispec.NewOptString("user-1"),
		Spec: apispec.SandboxTemplateSpec{
			Tags:         []string{},
			Sidecars:     []apispec.ContainerSpec{},
			AllowedTeams: []string{},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newSandboxVolume(id string) apispec.SandboxVolume {
	now := time.Now().UTC()
	return apispec.SandboxVolume{
		ID:         id,
		TeamID:     "team-1",
		UserID:     "user-1",
		CacheSize:  "1Gi",
		BufferSize: "1Mi",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func newSnapshot(id string) apispec.Snapshot {
	return apispec.Snapshot{
		ID:        id,
		VolumeID:  "vol-1",
		Name:      "snap-" + id,
		SizeBytes: 123,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func newContextResponse(id string) apispec.ContextResponse {
	return apispec.ContextResponse{
		ID:        id,
		Type:      apispec.ProcessTypeRepl,
		Running:   true,
		Paused:    false,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func newContextStatsResponse(id string) apispec.ContextStatsResponse {
	return apispec.ContextStatsResponse{
		ContextID: apispec.NewOptString(id),
		Type:      apispec.NewOptString("repl"),
		Running:   apispec.NewOptBool(true),
	}
}

func newMountResponse(volumeID, mountPoint string) apispec.MountResponse {
	return apispec.MountResponse{
		SandboxvolumeID: volumeID,
		MountPoint:      mountPoint,
		MountedAt:       time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func newNetworkPolicy() apispec.TplSandboxNetworkPolicy {
	return apispec.TplSandboxNetworkPolicy{
		Mode: apispec.TplSandboxNetworkPolicyModeAllowAll,
	}
}

func newFileInfo(name string) apispec.FileInfo {
	return apispec.FileInfo{
		Name:    apispec.NewOptString(name),
		Path:    apispec.NewOptString("/" + name),
		Type:    apispec.NewOptFileInfoType(apispec.FileInfoTypeFile),
		Size:    apispec.NewOptInt64(1),
		Mode:    apispec.NewOptString("0644"),
		ModTime: apispec.NewOptDateTime(time.Now().UTC()),
	}
}
