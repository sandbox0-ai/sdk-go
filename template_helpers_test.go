package sandbox0

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestNewTemplateCreateRequestBuildsTemplateSpec(t *testing.T) {
	t.Parallel()

	request := NewTemplateCreateRequest(
		"tpl-helper",
		TemplateMainContainer(
			"ubuntu:24.04",
			"4Gi",
			WithTemplateContainerEnv(apispec.EnvVar{Name: "APP_ENV", Value: "test"}),
		),
		WithTemplateDisplayName("Helper Template"),
		WithTemplateEnvVars(map[string]string{"MODE": "template"}),
		WithTemplateEmptyDirMount(TemplateEmptyDirMount("/var/lib/docker", "20Gi")),
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
	displayName, ok := request.Spec.DisplayName.Get()
	if !ok || displayName != "Helper Template" {
		t.Fatalf("displayName = %q, want Helper Template", displayName)
	}
	envVars, ok := request.Spec.EnvVars.Get()
	if !ok || envVars["MODE"] != "template" {
		t.Fatalf("envVars = %#v, want MODE=template", envVars)
	}
	pod, ok := request.Spec.Pod.Get()
	if !ok {
		t.Fatal("pod should be set")
	}
	if len(pod.EmptyDirMounts) != 1 {
		t.Fatalf("len(emptyDirMounts) = %d, want 1", len(pod.EmptyDirMounts))
	}
	if got := pod.EmptyDirMounts[0].MountPath; got != "/var/lib/docker" {
		t.Fatalf("emptyDirMounts[0].mountPath = %q, want /var/lib/docker", got)
	}
	if got, ok := pod.EmptyDirMounts[0].SizeLimit.Get(); !ok || got != "20Gi" {
		t.Fatalf("emptyDirMounts[0].sizeLimit = %q, want 20Gi", got)
	}
}

func TestNewTemplateFromSandboxCreateRequest(t *testing.T) {
	overrides := apispec.TemplateFromSandboxSpecOverrides{
		DisplayName: apispec.NewOptString("Python ready"),
		Pool: apispec.NewOptPoolStrategy(apispec.PoolStrategy{
			MinIdle: 1,
			MaxIdle: 2,
		}),
	}
	request := NewTemplateFromSandboxCreateRequest("python-ready", "sb_source", &overrides)
	if request.TemplateID != "python-ready" || request.SandboxID != "sb_source" {
		t.Fatalf("request = %+v", request)
	}
	got, ok := request.SpecOverrides.Get()
	if !ok || got.DisplayName.Or("") != "Python ready" {
		t.Fatalf("SpecOverrides = %+v, set = %v", got, ok)
	}
}

func TestWithTemplateEnvVarsCopiesMap(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{"MODE": "template"}
	spec := NewTemplateSpec(TemplateMainContainer("ubuntu:24.04", "4Gi"), WithTemplateEnvVars(envVars))
	envVars["MODE"] = "changed"

	copied, ok := spec.EnvVars.Get()
	if !ok {
		t.Fatal("spec.EnvVars should be set")
	}
	if copied["MODE"] != "template" {
		t.Fatalf("spec.EnvVars[MODE] = %q, want template", copied["MODE"])
	}
}
