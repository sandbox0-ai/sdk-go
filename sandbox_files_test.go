package sandbox0

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxFiles(t *testing.T) {
	sandboxID := "sb-1"
	filePath := "/hello.txt"
	listPath := "/api/v1/sandboxes/" + sandboxID + "/files/list"
	statPath := "/api/v1/sandboxes/" + sandboxID + "/files/stat"

	routes := routeMap{
		routeKey(http.MethodGet, "/api/v1/sandboxes/"+sandboxID+"/files"): func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello"))
		},
		routeKey(http.MethodGet, listPath): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessFileListResponse{
				Success: apispec.SuccessFileListResponseSuccessTrue,
				Data: apispec.NewOptSuccessFileListResponseData(apispec.SuccessFileListResponseData{
					Entries: []apispec.FileInfo{newFileInfo("hello.txt")},
				}),
			})
		},
		routeKey(http.MethodGet, statPath): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessFileStatResponse{
				Success: apispec.SuccessFileStatResponseSuccessTrue,
				Data:    apispec.NewOptFileInfo(newFileInfo("hello.txt")),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/files"): func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("mkdir") == "true" {
				writeJSON(t, w, http.StatusCreated, apispec.SuccessCreatedResponse{
					Success: apispec.SuccessCreatedResponseSuccessTrue,
					Data: apispec.NewOptSuccessCreatedResponseData(apispec.SuccessCreatedResponseData{
						Created: apispec.NewOptBool(true),
					}),
				})
				return
			}
			writeJSON(t, w, http.StatusOK, apispec.SuccessWrittenResponse{
				Success: apispec.SuccessWrittenResponseSuccessTrue,
				Data: apispec.NewOptSuccessWrittenResponseData(apispec.SuccessWrittenResponseData{
					Written: apispec.NewOptBool(true),
				}),
			})
		},
		routeKey(http.MethodDelete, "/api/v1/sandboxes/"+sandboxID+"/files"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessDeletedResponse{
				Success: apispec.SuccessDeletedResponseSuccessTrue,
				Data: apispec.NewOptSuccessDeletedResponseData(apispec.SuccessDeletedResponseData{
					Deleted: apispec.NewOptBool(true),
				}),
			})
		},
		routeKey(http.MethodPost, "/api/v1/sandboxes/"+sandboxID+"/files/move"): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, apispec.SuccessMovedResponse{
				Success: apispec.SuccessMovedResponseSuccessTrue,
				Data: apispec.NewOptSuccessMovedResponseData(apispec.SuccessMovedResponseData{
					Moved: apispec.NewOptBool(true),
				}),
			})
		},
	}

	server := newTestServer(t, routes)
	defer server.Close()
	client := newTestClient(t, server)
	sandbox := client.Sandbox(sandboxID)

	content, err := sandbox.ReadFile(context.Background(), filePath)
	if err != nil || string(content) != "hello" {
		t.Fatalf("read file failed: %v", err)
	}
	if _, err := sandbox.ListFiles(context.Background(), "/"); err != nil {
		t.Fatalf("list files failed: %v", err)
	}
	if _, err := sandbox.StatFile(context.Background(), filePath); err != nil {
		t.Fatalf("stat file failed: %v", err)
	}
	if _, err := sandbox.WriteFile(context.Background(), filePath, []byte("content")); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	if _, err := sandbox.Mkdir(context.Background(), "/tmp", true); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if _, err := sandbox.DeleteFile(context.Background(), filePath); err != nil {
		t.Fatalf("delete file failed: %v", err)
	}
	if _, err := sandbox.MoveFile(context.Background(), "/a", "/b"); err != nil {
		t.Fatalf("move file failed: %v", err)
	}
}

func TestDecodeFileGetResponseJSON(t *testing.T) {
	raw := []byte("hello")
	encoded := base64.StdEncoding.EncodeToString(raw)
	resp := &apispec.APIV1SandboxesIDFilesGetOKApplicationJSON{
		Success: apispec.APIV1SandboxesIDFilesGetOKApplicationJSONSuccessTrue,
		Data: apispec.NewOptFileContentResponse(apispec.FileContentResponse{
			Content:  apispec.NewOptString(encoded),
			Encoding: apispec.NewOptFileContentResponseEncoding(apispec.FileContentResponseEncodingBase64),
		}),
	}
	decoded, err := decodeFileGetResponse(resp)
	if err != nil {
		t.Fatalf("decode file response failed: %v", err)
	}
	if !bytes.Equal(decoded, raw) {
		t.Fatalf("unexpected decoded content: %q", string(decoded))
	}
}

func TestWatchFilesInvalidBaseURL(t *testing.T) {
	client := &Client{baseURL: "://bad"}
	sandbox := &Sandbox{ID: "sb-1", client: client}
	if _, _, _, err := sandbox.WatchFiles(context.Background(), "/", false); err == nil {
		t.Fatal("expected watch files error")
	}
}
