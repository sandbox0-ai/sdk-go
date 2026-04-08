package sandbox0

import (
	"encoding/json"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestTemplateCreateRequestSharedVolumeRoundTrip(t *testing.T) {
	t.Parallel()

	req := apispec.TemplateCreateRequest{
		TemplateID: "tpl-shared-volume",
		Spec: apispec.SandboxTemplateSpec{
			MainContainer: apispec.NewOptContainerSpec(apispec.ContainerSpec{
				Image: "nginx:1.27-alpine",
				Resources: apispec.ResourceQuota{
					CPU:    apispec.NewOptString("500m"),
					Memory: apispec.NewOptString("2Gi"),
				},
			}),
			SharedVolumes: []apispec.SharedVolumeSpec{{
				Name:      "workspace",
				MountPath: "/workspace/shared",
			}},
			Sidecars: []apispec.SidecarContainerSpec{{
				Name:    "helper",
				Image:   "busybox:latest",
				Command: []string{"sh", "-lc", "tail -f /dev/null"},
				Resources: apispec.ResourceQuota{
					CPU:    apispec.NewOptString("250m"),
					Memory: apispec.NewOptString("1Gi"),
				},
				Mounts: []apispec.ContainerMountSpec{{
					Name:      "workspace",
					MountPath: "/shared",
				}},
			}},
		},
	}

	encoded, err := (&req).MarshalJSON()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded apispec.TemplateCreateRequest
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(decoded.Spec.SharedVolumes) != 1 {
		t.Fatalf("len(decoded.Spec.SharedVolumes) = %d, want 1", len(decoded.Spec.SharedVolumes))
	}
	if decoded.Spec.SharedVolumes[0].Name != "workspace" {
		t.Fatalf("decoded shared volume name = %q, want workspace", decoded.Spec.SharedVolumes[0].Name)
	}
	if _, ok := decoded.Spec.SharedVolumes[0].SandboxVolumeId.Get(); ok {
		t.Fatal("decoded shared volume sandboxVolumeId should be unset")
	}
	if len(decoded.Spec.Sidecars) != 1 {
		t.Fatalf("len(decoded.Spec.Sidecars) = %d, want 1", len(decoded.Spec.Sidecars))
	}
	if len(decoded.Spec.Sidecars[0].Mounts) != 1 {
		t.Fatalf("len(decoded.Spec.Sidecars[0].Mounts) = %d, want 1", len(decoded.Spec.Sidecars[0].Mounts))
	}
	if decoded.Spec.Sidecars[0].Mounts[0].MountPath != "/shared" {
		t.Fatalf("decoded mount path = %q, want /shared", decoded.Spec.Sidecars[0].Mounts[0].MountPath)
	}
}

func TestNormalizeNullMapsHandlesSharedVolumes(t *testing.T) {
	t.Parallel()

	payload := any(map[string]any{
		"data": map[string]any{
			"spec": map[string]any{
				"sharedVolumes": nil,
				"sidecars":      nil,
			},
		},
	})

	if !normalizeNullMaps(&payload) {
		t.Fatal("normalizeNullMaps() changed = false, want true")
	}

	root := payload.(map[string]any)
	data := root["data"].(map[string]any)
	spec := data["spec"].(map[string]any)
	if sharedVolumes, ok := spec["sharedVolumes"].([]any); !ok || len(sharedVolumes) != 0 {
		t.Fatalf("sharedVolumes = %#v, want empty slice", spec["sharedVolumes"])
	}
	if sidecars, ok := spec["sidecars"].([]any); !ok || len(sidecars) != 0 {
		t.Fatalf("sidecars = %#v, want empty slice", spec["sidecars"])
	}
}
