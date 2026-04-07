//go:build e2e

package sandbox0_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestTemplateCRUD(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	templates, err := client.ListTemplate(ctx)
	if err != nil {
		t.Fatalf("list templates failed: %v", err)
	}
	if len(templates) == 0 {
		t.Fatalf("no templates available")
	}

	source := templates[0]
	sourceSpec := source.Spec
	sourceSpec.Sidecars = []apispec.SidecarContainerSpec{
		{
			Name:    "codex",
			Image:   "busybox:latest",
			Command: []string{"sh", "-lc", "touch /tmp/ready; tail -f /dev/null"},
			Resources: apispec.ResourceQuota{
				CPU:    apispec.NewOptString("250m"),
				Memory: apispec.NewOptString("1Gi"),
			},
			ReadinessProbe: apispec.NewOptProbe(apispec.Probe{
				Exec:                apispec.NewOptExecAction(apispec.ExecAction{Command: []string{"test", "-f", "/tmp/ready"}}),
				InitialDelaySeconds: apispec.NewOptInt32(1),
				PeriodSeconds:       apispec.NewOptInt32(1),
				FailureThreshold:    apispec.NewOptInt32(1),
			}),
		},
	}
	templateID := fmt.Sprintf("sdk-e2e-%d", time.Now().UnixNano())

	createReq := apispec.TemplateCreateRequest{
		TemplateID: templateID,
		Spec:       sourceSpec,
	}
	created, err := client.CreateTemplate(ctx, createReq)
	if err != nil {
		t.Fatalf("create template failed: %v", err)
	}
	if created == nil || created.TemplateID == "" {
		t.Fatalf("create template returned empty template")
	}
	if len(created.Spec.Sidecars) != 1 {
		t.Fatalf("created template sidecars = %d, want 1", len(created.Spec.Sidecars))
	}
	if !created.Spec.Sidecars[0].ReadinessProbe.IsSet() {
		t.Fatalf("created template sidecar readiness probe missing")
	}
	deleted := false
	t.Cleanup(func() {
		if deleted {
			return
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteTemplate(cleanupCtx, templateID)
	})

	fetched, err := client.GetTemplate(ctx, templateID)
	if err != nil {
		t.Fatalf("get template failed: %v", err)
	}
	if len(fetched.Spec.Sidecars) != 1 {
		t.Fatalf("fetched template sidecars = %d, want 1", len(fetched.Spec.Sidecars))
	}
	if !fetched.Spec.Sidecars[0].ReadinessProbe.IsSet() {
		t.Fatalf("fetched template sidecar readiness probe missing")
	}

	updatedSpec := fetched.Spec
	updatedSpec.DisplayName = apispec.NewOptString("SDK E2E Updated")
	updateReq := apispec.TemplateUpdateRequest{Spec: updatedSpec}
	updated, err := client.UpdateTemplate(ctx, templateID, updateReq)
	if err != nil {
		t.Fatalf("update template failed: %v", err)
	}
	if len(updated.Spec.Sidecars) != 1 {
		t.Fatalf("updated template sidecars = %d, want 1", len(updated.Spec.Sidecars))
	}

	if _, err := client.DeleteTemplate(ctx, templateID); err != nil {
		t.Fatalf("delete template failed: %v", err)
	}
	deleted = true
}
