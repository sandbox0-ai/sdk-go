package sandbox0

import (
	"context"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestClientTemplates(t *testing.T) {
	templateID := "default"
	template := newTemplate(templateID)
	routes := routeMap{
		routeKey(http.MethodGet, "/api/v1/templates"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessTemplateListResponse{
				Success: apispec.SuccessTemplateListResponseSuccessTrue,
				Data: apispec.NewOptSuccessTemplateListResponseData(apispec.SuccessTemplateListResponseData{
					Templates: []apispec.Template{template},
				}),
			})
		},
		routeKey(http.MethodGet, "/api/v1/templates/"+templateID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessTemplateResponse{
				Success: apispec.SuccessTemplateResponseSuccessTrue,
				Data:    apispec.NewOptTemplate(template),
			})
		},
		routeKey(http.MethodPost, "/api/v1/templates"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, apispec.SuccessTemplateResponse{
				Success: apispec.SuccessTemplateResponseSuccessTrue,
				Data:    apispec.NewOptTemplate(template),
			})
		},
		routeKey(http.MethodPut, "/api/v1/templates/"+templateID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessTemplateResponse{
				Success: apispec.SuccessTemplateResponseSuccessTrue,
				Data:    apispec.NewOptTemplate(template),
			})
		},
		routeKey(http.MethodDelete, "/api/v1/templates/"+templateID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessMessageResponse{
				Success: apispec.SuccessMessageResponseSuccessTrue,
				Data: apispec.NewOptSuccessMessageResponseData(apispec.SuccessMessageResponseData{
					Message: apispec.NewOptString("deleted"),
				}),
			})
		},
	}

	server := newTestServer(t, routes)
	defer server.Close()
	client := newTestClient(t, server)

	list, err := client.ListTemplate(context.Background())
	if err != nil || len(list) == 0 {
		t.Fatalf("list templates failed: %v", err)
	}
	if _, err := client.GetTemplate(context.Background(), templateID); err != nil {
		t.Fatalf("get template failed: %v", err)
	}
	if _, err := client.CreateTemplate(context.Background(), apispec.TemplateCreateRequest{TemplateID: templateID}); err != nil {
		t.Fatalf("create template failed: %v", err)
	}
	if _, err := client.UpdateTemplate(context.Background(), templateID, apispec.TemplateUpdateRequest{}); err != nil {
		t.Fatalf("update template failed: %v", err)
	}
	if _, err := client.DeleteTemplate(context.Background(), templateID); err != nil {
		t.Fatalf("delete template failed: %v", err)
	}
}
