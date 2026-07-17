package sandbox0

import (
	"encoding/json"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestTemplateCreateRequestEnvVarsRoundTrip(t *testing.T) {
	t.Parallel()

	req := apispec.TemplateCreateRequest{
		TemplateID: "tpl-env-vars",
		Spec: apispec.SandboxTemplateSpec{
			MainContainer: apispec.NewOptContainerSpec(apispec.ContainerSpec{
				Image: "nginx:1.27-alpine",
				Resources: apispec.ResourceQuota{
					Memory: "2Gi",
				},
			}),
			EnvVars: apispec.NewOptSandboxTemplateSpecEnvVars(apispec.SandboxTemplateSpecEnvVars{"MODE": "template"}),
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

	envVars, ok := decoded.Spec.EnvVars.Get()
	if !ok || envVars["MODE"] != "template" {
		t.Fatalf("decoded envVars = %#v, want MODE=template", envVars)
	}
}

func TestNormalizeNullMapsHandlesTemplateTags(t *testing.T) {
	t.Parallel()

	payload := any(map[string]any{
		"data": map[string]any{
			"spec": map[string]any{
				"tags": nil,
			},
		},
	})

	if !normalizeNullMaps(&payload) {
		t.Fatal("normalizeNullMaps() changed = false, want true")
	}

	root := payload.(map[string]any)
	data := root["data"].(map[string]any)
	spec := data["spec"].(map[string]any)
	if tags, ok := spec["tags"].([]any); !ok || len(tags) != 0 {
		t.Fatalf("tags = %#v, want empty slice", spec["tags"])
	}
}
