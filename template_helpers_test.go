package sandbox0

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestNewTemplateCreateRequestBuildsSharedVolumeSpec(t *testing.T) {
	t.Parallel()

	request := NewTemplateCreateRequest(
		"tpl-helper",
		TemplateMainContainer(
			"ubuntu:24.04",
			"1",
			"4Gi",
			WithTemplateContainerEnv(apispec.EnvVar{Name: "APP_ENV", Value: "test"}),
		),
		WithTemplateDisplayName("Helper Template"),
		WithTemplateSharedVolume(
			TemplateSharedVolume(
				"workspace",
				"vol_123",
				"/workspace/shared",
				WithTemplateSharedVolumeWriteback(true),
			),
		),
		WithTemplateSidecar(
			TemplateSidecar(
				"helper",
				"busybox:latest",
				"250m",
				"1Gi",
				WithTemplateSidecarCommand("sh", "-lc", "tail -f /dev/null"),
				WithTemplateSidecarMount(TemplateMount("workspace", "/shared")),
			),
		),
	)

	if request.TemplateID != "tpl-helper" {
		t.Fatalf("TemplateID = %q, want tpl-helper", request.TemplateID)
	}
	main, ok := request.Spec.MainContainer.Get()
	if !ok {
		t.Fatal("mainContainer should be set")
	}
	if main.Image != "ubuntu:24.04" {
		t.Fatalf("mainContainer.image = %q, want ubuntu:24.04", main.Image)
	}
	if len(request.Spec.SharedVolumes) != 1 {
		t.Fatalf("len(sharedVolumes) = %d, want 1", len(request.Spec.SharedVolumes))
	}
	if request.Spec.SharedVolumes[0].Name != "workspace" {
		t.Fatalf("sharedVolumes[0].name = %q, want workspace", request.Spec.SharedVolumes[0].Name)
	}
	writeback, ok := request.Spec.SharedVolumes[0].Writeback.Get()
	if !ok || !writeback {
		t.Fatalf("sharedVolumes[0].writeback = %v, want true", writeback)
	}
	if len(request.Spec.Sidecars) != 1 {
		t.Fatalf("len(sidecars) = %d, want 1", len(request.Spec.Sidecars))
	}
	if len(request.Spec.Sidecars[0].Mounts) != 1 {
		t.Fatalf("len(sidecars[0].mounts) = %d, want 1", len(request.Spec.Sidecars[0].Mounts))
	}
	if request.Spec.Sidecars[0].Mounts[0].MountPath != "/shared" {
		t.Fatalf("sidecars[0].mounts[0].mountPath = %q, want /shared", request.Spec.Sidecars[0].Mounts[0].MountPath)
	}
	displayName, ok := request.Spec.DisplayName.Get()
	if !ok || displayName != "Helper Template" {
		t.Fatalf("displayName = %q, want Helper Template", displayName)
	}
}
