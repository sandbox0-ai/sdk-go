package sandbox0

import (
	"context"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxContexts(t *testing.T) {
	sandboxID := "sb-1"
	contextID := "ctx-1"
	ctxResp := newContextResponse(contextID)

	routes := routeMap{
		routeKey(http.MethodGet, "/api/v1/sandboxes/"+sandboxID+"/contexts"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessContextListResponse{
				Success: apispec.SuccessContextListResponseSuccessTrue,
				Data: apispec.NewOptSuccessContextListResponseData(apispec.SuccessContextListResponseData{
					Contexts: []apispec.ContextResponse{ctxResp},
				}),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/contexts"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, apispec.SuccessContextResponse{
				Success: apispec.SuccessContextResponseSuccessTrue,
				Data:    apispec.NewOptContextResponse(ctxResp),
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/"+sandboxID+"/contexts/"+contextID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessContextResponse{
				Success: apispec.SuccessContextResponseSuccessTrue,
				Data:    apispec.NewOptContextResponse(ctxResp),
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxes/"+sandboxID+"/contexts/"+contextID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessDeletedResponse{
				Success: apispec.SuccessDeletedResponseSuccessTrue,
				Data: apispec.NewOptSuccessDeletedResponseData(apispec.SuccessDeletedResponseData{
					Deleted: apispec.NewOptBool(true),
				}),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/contexts/"+contextID+"/restart"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessContextResponse{
				Success: apispec.SuccessContextResponseSuccessTrue,
				Data:    apispec.NewOptContextResponse(ctxResp),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/contexts/"+contextID+"/input"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessWrittenResponse{
				Success: apispec.SuccessWrittenResponseSuccessTrue,
				Data: apispec.NewOptSuccessWrittenResponseData(apispec.SuccessWrittenResponseData{
					Written: apispec.NewOptBool(true),
				}),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/contexts/"+contextID+"/exec"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessContextExecResponse{
				Success: apispec.SuccessContextExecResponseSuccessTrue,
				Data: apispec.NewOptContextExecResponse(apispec.ContextExecResponse{
					Output: "ok",
				}),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/contexts/"+contextID+"/resize"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessResizedResponse{
				Success: apispec.SuccessResizedResponseSuccessTrue,
				Data: apispec.NewOptSuccessResizedResponseData(apispec.SuccessResizedResponseData{
					Resized: apispec.NewOptBool(true),
				}),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/contexts/"+contextID+"/signal"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessSignaledResponse{
				Success: apispec.SuccessSignaledResponseSuccessTrue,
				Data: apispec.NewOptSuccessSignaledResponseData(apispec.SuccessSignaledResponseData{
					Signaled: apispec.NewOptBool(true),
				}),
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/"+sandboxID+"/contexts/"+contextID+"/stats"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessContextStatsResponse{
				Success: apispec.SuccessContextStatsResponseSuccessTrue,
				Data:    apispec.NewOptContextStatsResponse(newContextStatsResponse(contextID)),
			})
		},
	}

	server := newTestServer(t, routes)
	defer server.Close()
	client := newTestClient(t, server)
	sandbox := client.Sandbox(sandboxID)

	if _, err := sandbox.ListContext(context.Background()); err != nil {
		t.Fatalf("list contexts failed: %v", err)
	}
	if _, err := sandbox.CreateContext(context.Background(), apispec.CreateContextRequest{}); err != nil {
		t.Fatalf("create context failed: %v", err)
	}
	if _, err := sandbox.GetContext(context.Background(), contextID); err != nil {
		t.Fatalf("get context failed: %v", err)
	}
	if _, err := sandbox.RestartContext(context.Background(), contextID); err != nil {
		t.Fatalf("restart context failed: %v", err)
	}
	if _, err := sandbox.ContextInput(context.Background(), contextID, "in"); err != nil {
		t.Fatalf("context input failed: %v", err)
	}
	if _, err := sandbox.ContextExec(context.Background(), contextID, "exec"); err != nil {
		t.Fatalf("context exec failed: %v", err)
	}
	if _, err := sandbox.ContextResize(context.Background(), contextID, 10, 20); err != nil {
		t.Fatalf("context resize failed: %v", err)
	}
	if _, err := sandbox.ContextSignal(context.Background(), contextID, "TERM"); err != nil {
		t.Fatalf("context signal failed: %v", err)
	}
	if _, err := sandbox.ContextStats(context.Background(), contextID); err != nil {
		t.Fatalf("context stats failed: %v", err)
	}
	if _, err := sandbox.DeleteContext(context.Background(), contextID); err != nil {
		t.Fatalf("delete context failed: %v", err)
	}
}
