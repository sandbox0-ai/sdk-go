package sandbox0

import (
	"context"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxVolumes(t *testing.T) {
	sandboxID := "sb-1"
	volumeID := "vol-1"
	mountPoint := "/data"
	mount := newMountResponse(volumeID, mountPoint)

	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/sandboxvolumes/mount"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessMountResponse{
				Success: apispec.SuccessMountResponseSuccessTrue,
				Data:    apispec.NewOptMountResponse(mount),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/sandboxvolumes/unmount"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessUnmountedResponse{
				Success: apispec.SuccessUnmountedResponseSuccessTrue,
				Data: apispec.NewOptSuccessUnmountedResponseData(apispec.SuccessUnmountedResponseData{
					Unmounted: apispec.NewOptBool(true),
				}),
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxes/"+sandboxID+"/sandboxvolumes/status"): func(w http.ResponseWriter, r *http.Request) {
			status := apispec.MountStatus{
				SandboxvolumeID: apispec.NewOptString(volumeID),
				MountPoint:      apispec.NewOptString(mountPoint),
			}
			writeJSON(t, w, http.StatusOK, apispec.SuccessMountStatusResponse{
				Success: apispec.SuccessMountStatusResponseSuccessTrue,
				Data: apispec.NewOptSuccessMountStatusResponseData(apispec.SuccessMountStatusResponseData{
					Mounts: []apispec.MountStatus{status},
				}),
			})
		},
	}

	server := newTestServer(t, routes)
	defer server.Close()
	client := newTestClient(t, server)
	sandbox := client.Sandbox(sandboxID)

	if _, err := sandbox.Mount(context.Background(), volumeID, mountPoint, nil); err != nil {
		t.Fatalf("mount failed: %v", err)
	}
	if _, err := sandbox.MountStatus(context.Background()); err != nil {
		t.Fatalf("mount status failed: %v", err)
	}
	if _, err := sandbox.Unmount(context.Background(), volumeID); err != nil {
		t.Fatalf("unmount failed: %v", err)
	}
}
