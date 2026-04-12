package sandbox0

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestNewTemplateCreateRequestBuildsWarmProcessSpec(t *testing.T) {
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
		WithTemplateWarmProcess(
			TemplateWarmProcess(
				apispec.WarmProcessSpecTypeCmd,
				WithTemplateWarmProcessAlias("helper"),
				WithTemplateWarmProcessCommand("sh", "-lc", "tail -f /dev/null"),
				WithTemplateWarmProcessCWD("/workspace"),
				WithTemplateWarmProcessEnvVars(map[string]string{"MODE": "warm"}),
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
	if len(request.Spec.WarmProcesses) != 1 {
		t.Fatalf("len(warmProcesses) = %d, want 1", len(request.Spec.WarmProcesses))
	}
	process := request.Spec.WarmProcesses[0]
	if process.Type != apispec.WarmProcessSpecTypeCmd {
		t.Fatalf("warmProcesses[0].type = %q, want cmd", process.Type)
	}
	alias, ok := process.Alias.Get()
	if !ok || alias != "helper" {
		t.Fatalf("warmProcesses[0].alias = %q, want helper", alias)
	}
	if len(process.Command) != 3 || process.Command[2] != "tail -f /dev/null" {
		t.Fatalf("warmProcesses[0].command = %#v, want shell command", process.Command)
	}
	cwd, ok := process.Cwd.Get()
	if !ok || cwd != "/workspace" {
		t.Fatalf("warmProcesses[0].cwd = %q, want /workspace", cwd)
	}
	envVars, ok := process.EnvVars.Get()
	if !ok || envVars["MODE"] != "warm" {
		t.Fatalf("warmProcesses[0].envVars = %#v, want MODE=warm", envVars)
	}
	displayName, ok := request.Spec.DisplayName.Get()
	if !ok || displayName != "Helper Template" {
		t.Fatalf("displayName = %q, want Helper Template", displayName)
	}
}

func TestTemplateWarmProcessCopiesEnvVars(t *testing.T) {
	t.Parallel()

	envVars := map[string]string{"MODE": "warm"}
	process := TemplateWarmProcess(
		apispec.WarmProcessSpecTypeRepl,
		WithTemplateWarmProcessAlias("shell"),
		WithTemplateWarmProcessEnvVars(envVars),
	)
	envVars["MODE"] = "changed"

	copied, ok := process.EnvVars.Get()
	if !ok {
		t.Fatal("process.EnvVars should be set")
	}
	if copied["MODE"] != "warm" {
		t.Fatalf("process.EnvVars[MODE] = %q, want warm", copied["MODE"])
	}
}
