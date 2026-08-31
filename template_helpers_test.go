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
			"docker.io/library/ubuntu@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"4Gi",
			WithTemplateContainerEnv(apispec.EnvVar{Name: "APP_ENV", Value: "test"}),
			WithTemplateContainerSecurityClass(apispec.ContainerSpecSecurityClassPrivileged),
		),
		WithTemplateDisplayName("Helper Template"),
		WithTemplateEnvVars(map[string]string{"MODE": "template"}),
		WithTemplateEphemeralMount(TemplateEphemeralMount("/var/lib/docker", "20Gi")),
	)

	if request.TemplateID != "tpl-helper" {
		t.Fatalf("TemplateID = %q, want tpl-helper", request.TemplateID)
	}
	main := request.Spec.MainContainer
	if main.Image != "docker.io/library/ubuntu@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("mainContainer.image = %q", main.Image)
	}
	if got, ok := main.SecurityClass.Get(); !ok || got != apispec.ContainerSpecSecurityClassPrivileged {
		t.Fatalf("mainContainer.securityClass = %q, want privileged", got)
	}
	displayName, ok := request.Spec.DisplayName.Get()
	if !ok || displayName != "Helper Template" {
		t.Fatalf("displayName = %q, want Helper Template", displayName)
	}
	envVars, ok := request.Spec.EnvVars.Get()
	if !ok || envVars["MODE"] != "template" {
		t.Fatalf("envVars = %#v, want MODE=template", envVars)
	}
	if len(request.Spec.EphemeralMounts) != 1 {
		t.Fatalf("len(ephemeralMounts) = %d, want 1", len(request.Spec.EphemeralMounts))
	}
	if got := request.Spec.EphemeralMounts[0].MountPath; got != "/var/lib/docker" {
		t.Fatalf("ephemeralMounts[0].mountPath = %q, want /var/lib/docker", got)
	}
	if got := request.Spec.EphemeralMounts[0].SizeLimit; got != "20Gi" {
		t.Fatalf("ephemeralMounts[0].sizeLimit = %q, want 20Gi", got)
	}
}

func TestNewTemplateFromSandboxCreateRequest(t *testing.T) {
	overrides := apispec.TemplateFromSandboxSpecOverrides{
		DisplayName: apispec.NewOptString("Python ready"),
		Tags:        []string{"python"},
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
	spec := NewTemplateSpec(
		TemplateMainContainer("docker.io/library/ubuntu@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "4Gi"),
		WithTemplateEnvVars(envVars),
	)
	envVars["MODE"] = "changed"

	copied, ok := spec.EnvVars.Get()
	if !ok {
		t.Fatal("spec.EnvVars should be set")
	}
	if copied["MODE"] != "template" {
		t.Fatalf("spec.EnvVars[MODE] = %q, want template", copied["MODE"])
	}
}
