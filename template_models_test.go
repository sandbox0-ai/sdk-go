package sandbox0

import (
	"encoding/json"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestTemplateCreateRequestWarmProcessRoundTrip(t *testing.T) {
	t.Parallel()

	req := apispec.TemplateCreateRequest{
		TemplateID: "tpl-warm-process",
		Spec: apispec.SandboxTemplateSpec{
			MainContainer: apispec.NewOptContainerSpec(apispec.ContainerSpec{
				Image: "nginx:1.27-alpine",
				Resources: apispec.ResourceQuota{
					CPU:    apispec.NewOptString("500m"),
					Memory: apispec.NewOptString("2Gi"),
				},
			}),
			WarmProcesses: []apispec.WarmProcessSpec{{
				Type:    apispec.WarmProcessSpecTypeCmd,
				Alias:   apispec.NewOptString("helper"),
				Command: []string{"sh", "-lc", "tail -f /dev/null"},
				Cwd:     apispec.NewOptString("/workspace"),
				EnvVars: apispec.NewOptWarmProcessSpecEnvVars(apispec.WarmProcessSpecEnvVars{"MODE": "warm"}),
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

	if len(decoded.Spec.WarmProcesses) != 1 {
		t.Fatalf("len(decoded.Spec.WarmProcesses) = %d, want 1", len(decoded.Spec.WarmProcesses))
	}
	process := decoded.Spec.WarmProcesses[0]
	if process.Type != apispec.WarmProcessSpecTypeCmd {
		t.Fatalf("decoded warm process type = %q, want cmd", process.Type)
	}
	if len(process.Command) != 3 || process.Command[2] != "tail -f /dev/null" {
		t.Fatalf("decoded warm process command = %#v, want shell command", process.Command)
	}
	envVars, ok := process.EnvVars.Get()
	if !ok || envVars["MODE"] != "warm" {
		t.Fatalf("decoded warm process envVars = %#v, want MODE=warm", envVars)
	}
}

func TestNormalizeNullMapsHandlesWarmProcesses(t *testing.T) {
	t.Parallel()

	payload := any(map[string]any{
		"data": map[string]any{
			"spec": map[string]any{
				"warmProcesses": nil,
			},
		},
	})

	if !normalizeNullMaps(&payload) {
		t.Fatal("normalizeNullMaps() changed = false, want true")
	}

	root := payload.(map[string]any)
	data := root["data"].(map[string]any)
	spec := data["spec"].(map[string]any)
	if warmProcesses, ok := spec["warmProcesses"].([]any); !ok || len(warmProcesses) != 0 {
		t.Fatalf("warmProcesses = %#v, want empty slice", spec["warmProcesses"])
	}
}
