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
		if len(body.Mounts) != 2 {
			t.Fatalf("mount count = %d, want 2", len(body.Mounts))
		}
		if body.Mounts[0].SandboxvolumeID != "vol_1" || body.Mounts[0].MountPoint != "/workspace/data" {
			t.Fatalf("mount[0] = %+v, want vol_1:/workspace/data", body.Mounts[0])
		}
		if body.Mounts[1].SandboxvolumeID != "vol_2" || body.Mounts[1].MountPoint != "/workspace/readonly" {
			t.Fatalf("mount[1] = %+v, want vol_2:/workspace/readonly", body.Mounts[1])
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
		WithSandboxBootstrapMount("vol_1", "/workspace/data"),
		WithSandboxBootstrapMounts(SandboxBootstrapMount{
			SandboxVolumeID: "vol_2",
			MountPoint:      "/workspace/readonly",
		}),
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
