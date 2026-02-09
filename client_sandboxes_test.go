package sandbox0

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestClientSandboxes(t *testing.T) {
	sandboxID := "sb-1"
	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxes"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, apispec.SuccessClaimResponse{
				Success: apispec.SuccessClaimResponseSuccessTrue,
				Data:    apispec.NewOptClaimResponse(newClaimResponse(sandboxID)),
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/"+sandboxID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessSandboxResponse{
				Success: apispec.SuccessSandboxResponseSuccessTrue,
				Data:    apispec.NewOptSandbox(newSandbox(sandboxID)),
			})
		},
		routeKey(http.MethodPatch, "/api/v1/sandboxes/"+sandboxID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessSandboxResponse{
				Success: apispec.SuccessSandboxResponseSuccessTrue,
				Data:    apispec.NewOptSandbox(newSandbox(sandboxID)),
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxes/"+sandboxID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessMessageResponse{
				Success: apispec.SuccessMessageResponseSuccessTrue,
				Data: apispec.NewOptSuccessMessageResponseData(apispec.SuccessMessageResponseData{
					Message: apispec.NewOptString("deleted"),
				}),
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/"+sandboxID+"/status"): func(w http.ResponseWriter, r *http.Request) {
			status := apispec.SandboxStatus{
				SandboxID: apispec.NewOptString(sandboxID),
				Status:    apispec.NewOptString("running"),
			}
			writeJSON(t, w, http.StatusOK, apispec.SuccessSandboxStatusResponse{
				Success: apispec.SuccessSandboxStatusResponseSuccessTrue,
				Data:    apispec.NewOptSandboxStatus(status),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/pause"): func(w http.ResponseWriter, r *http.Request) {
			payload := apispec.PauseSandboxResponse{
				SandboxID: sandboxID,
				Paused:    true,
			}
			writeJSON(t, w, http.StatusOK, apispec.SuccessPauseSandboxResponse{
				Success: apispec.SuccessPauseSandboxResponseSuccessTrue,
				Data:    apispec.NewOptPauseSandboxResponse(payload),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/resume"): func(w http.ResponseWriter, r *http.Request) {
			payload := apispec.ResumeSandboxResponse{
				SandboxID: sandboxID,
				Resumed:   true,
			}
			writeJSON(t, w, http.StatusOK, apispec.SuccessResumeSandboxResponse{
				Success: apispec.SuccessResumeSandboxResponseSuccessTrue,
				Data:    apispec.NewOptResumeSandboxResponse(payload),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/refresh"): func(w http.ResponseWriter, r *http.Request) {
			payload := apispec.RefreshResponse{
				SandboxID: sandboxID,
				ExpiresAt: time.Now().UTC(),
			}
			writeJSON(t, w, http.StatusOK, apispec.SuccessRefreshResponse{
				Success: apispec.SuccessRefreshResponseSuccessTrue,
				Data:    apispec.NewOptRefreshResponse(payload),
			})
		},
	}

	server := newTestServer(t, routes)
	defer server.Close()
	client := newTestClient(t, server)

	sandbox, err := client.ClaimSandbox(context.Background(), "default")
	if err != nil || sandbox == nil || sandbox.ID != sandboxID {
		t.Fatalf("claim sandbox failed: %v", err)
	}

	if _, err := client.GetSandbox(context.Background(), sandboxID); err != nil {
		t.Fatalf("get sandbox failed: %v", err)
	}
	if _, err := client.UpdateSandbox(context.Background(), sandboxID, apispec.SandboxUpdateRequest{}); err != nil {
		t.Fatalf("update sandbox failed: %v", err)
	}
	if _, err := client.StatusSandbox(context.Background(), sandboxID); err != nil {
		t.Fatalf("status sandbox failed: %v", err)
	}
	if _, err := client.PauseSandbox(context.Background(), sandboxID); err != nil {
		t.Fatalf("pause sandbox failed: %v", err)
	}
	if _, err := client.ResumeSandbox(context.Background(), sandboxID); err != nil {
		t.Fatalf("resume sandbox failed: %v", err)
	}
	if _, err := client.RefreshSandbox(context.Background(), sandboxID, nil); err != nil {
		t.Fatalf("refresh sandbox failed: %v", err)
	}
	if _, err := client.DeleteSandbox(context.Background(), sandboxID); err != nil {
		t.Fatalf("delete sandbox failed: %v", err)
	}
}
