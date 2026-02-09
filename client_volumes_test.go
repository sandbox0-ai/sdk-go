package sandbox0

import (
	"context"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestClientVolumes(t *testing.T) {
	volumeID := "vol-1"
	snapshotID := "snap-1"
	volume := newSandboxVolume(volumeID)
	snapshot := newSnapshot(snapshotID)

	routes := routeMap{
		routeKey(http.MethodPost, "/api/v1/sandboxvolumes"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, apispec.SuccessSandboxVolumeResponse{
				Success: apispec.SuccessSandboxVolumeResponseSuccessTrue,
				Data:    apispec.NewOptSandboxVolume(volume),
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxvolumes"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessSandboxVolumeListResponse{
				Success: apispec.SuccessSandboxVolumeListResponseSuccessTrue,
				Data:    []apispec.SandboxVolume{volume},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxvolumes/"+volumeID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessSandboxVolumeResponse{
				Success: apispec.SuccessSandboxVolumeResponseSuccessTrue,
				Data:    apispec.NewOptSandboxVolume(volume),
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxvolumes/"+volumeID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessDeletedResponse{
				Success: apispec.SuccessDeletedResponseSuccessTrue,
				Data: apispec.NewOptSuccessDeletedResponseData(apispec.SuccessDeletedResponseData{
					Deleted: apispec.NewOptBool(true),
				}),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxvolumes/"+volumeID+"/snapshots"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusCreated, apispec.SuccessSnapshotResponse{
				Success: apispec.SuccessSnapshotResponseSuccessTrue,
				Data:    apispec.NewOptSnapshot(snapshot),
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxvolumes/"+volumeID+"/snapshots"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessSnapshotListResponse{
				Success: apispec.SuccessSnapshotListResponseSuccessTrue,
				Data:    []apispec.Snapshot{snapshot},
			})
		},
		routeKey(http.MethodGet, "/api/v1/sandboxvolumes/"+volumeID+"/snapshots/"+snapshotID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessSnapshotResponse{
				Success: apispec.SuccessSnapshotResponseSuccessTrue,
				Data:    apispec.NewOptSnapshot(snapshot),
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxvolumes/"+volumeID+"/snapshots/"+snapshotID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessDeletedResponse{
				Success: apispec.SuccessDeletedResponseSuccessTrue,
				Data: apispec.NewOptSuccessDeletedResponseData(apispec.SuccessDeletedResponseData{
					Deleted: apispec.NewOptBool(true),
				}),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxvolumes/"+volumeID+"/snapshots/"+snapshotID+"/restore"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessRestoreResponse{
				Success: apispec.SuccessRestoreResponseSuccessTrue,
				Data: apispec.NewOptSuccessRestoreResponseData(apispec.SuccessRestoreResponseData{
					Status: apispec.NewOptString("restored"),
				}),
			})
		},
	}

	server := newTestServer(t, routes)
	defer server.Close()
	client := newTestClient(t, server)

	if _, err := client.CreateVolume(context.Background(), apispec.CreateSandboxVolumeRequest{}); err != nil {
		t.Fatalf("create volume failed: %v", err)
	}
	if _, err := client.ListVolume(context.Background()); err != nil {
		t.Fatalf("list volumes failed: %v", err)
	}
	if _, err := client.GetVolume(context.Background(), volumeID); err != nil {
		t.Fatalf("get volume failed: %v", err)
	}
	if _, err := client.DeleteVolume(context.Background(), volumeID); err != nil {
		t.Fatalf("delete volume failed: %v", err)
	}
	if _, err := client.CreateVolumeSnapshot(context.Background(), volumeID, apispec.CreateSnapshotRequest{Name: "snap"}); err != nil {
		t.Fatalf("create snapshot failed: %v", err)
	}
	if _, err := client.ListVolumeSnapshots(context.Background(), volumeID); err != nil {
		t.Fatalf("list snapshots failed: %v", err)
	}
	if _, err := client.GetVolumeSnapshot(context.Background(), volumeID, snapshotID); err != nil {
		t.Fatalf("get snapshot failed: %v", err)
	}
	if _, err := client.DeleteVolumeSnapshot(context.Background(), volumeID, snapshotID); err != nil {
		t.Fatalf("delete snapshot failed: %v", err)
	}
	if _, err := client.RestoreVolumeSnapshot(context.Background(), volumeID, snapshotID); err != nil {
		t.Fatalf("restore snapshot failed: %v", err)
	}
}
