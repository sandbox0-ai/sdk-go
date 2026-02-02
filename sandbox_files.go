package sandbox0

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// FileWatchSubscribeRequest is a subscribe message for file watch.
type FileWatchSubscribeRequest struct {
	Action    string `json:"action"`
	Path      string `json:"path"`
	Recursive bool   `json:"recursive,omitempty"`
}

// FileWatchUnsubscribeRequest is an unsubscribe message for file watch.
type FileWatchUnsubscribeRequest struct {
	Action  string `json:"action"`
	WatchID string `json:"watch_id"`
}

// FileWatchResponse represents a server watch message.
type FileWatchResponse struct {
	Type    string `json:"type"`
	WatchID string `json:"watch_id,omitempty"`
	Event   string `json:"event,omitempty"`
	Path    string `json:"path,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Read reads a file and returns raw bytes.
func (s *Sandbox) ReadFile(ctx context.Context, path string) ([]byte, error) {
	params := apispec.GetApiV1SandboxesIdFilesParams{
		Path: apispec.FilePath(path),
	}
	resp, err := s.client.api.GetApiV1SandboxesIdFilesWithResponse(ctx, apispec.SandboxID(s.ID), &params)
	if err != nil {
		return nil, err
	}
	if resp.HTTPResponse == nil || resp.HTTPResponse.StatusCode != http.StatusOK {
		return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
	}
	return resp.Body, nil
}

// Stat retrieves file metadata.
func (s *Sandbox) StatFile(ctx context.Context, path string) (*apispec.FileInfo, error) {
	params := apispec.GetApiV1SandboxesIdFilesStatParams{
		Path: apispec.FilePath(path),
	}
	resp, err := s.client.api.GetApiV1SandboxesIdFilesStatWithResponse(ctx, apispec.SandboxID(s.ID), &params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200.Data, nil
}

// List returns directory entries.
func (s *Sandbox) ListFiles(ctx context.Context, path string) ([]apispec.FileInfo, error) {
	params := apispec.GetApiV1SandboxesIdFilesListParams{
		Path: apispec.FilePath(path),
	}
	resp, err := s.client.api.GetApiV1SandboxesIdFilesListWithResponse(ctx, apispec.SandboxID(s.ID), &params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil || resp.JSON200.Data == nil || resp.JSON200.Data.Entries == nil {
		return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
	}
	return *resp.JSON200.Data.Entries, nil
}

// ReadBinary reads file content as base64 and decodes it.
func (s *Sandbox) ReadBinaryFile(ctx context.Context, path string) ([]byte, error) {
	params := apispec.GetApiV1SandboxesIdFilesBinaryParams{
		Path: apispec.FilePath(path),
	}
	resp, err := s.client.api.GetApiV1SandboxesIdFilesBinaryWithResponse(ctx, apispec.SandboxID(s.ID), &params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil || resp.JSON200.Data == nil || resp.JSON200.Data.Content == nil {
		return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
	}
	content := *resp.JSON200.Data.Content
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

// Write writes a file.
func (s *Sandbox) WriteFile(ctx context.Context, path string, data []byte) (*apispec.SuccessWrittenResponse, error) {
	body := bytes.NewReader(data)
	params := apispec.PostApiV1SandboxesIdFilesParams{
		Path: apispec.FilePath(path),
	}
	resp, err := s.client.api.PostApiV1SandboxesIdFilesWithBodyWithResponse(
		ctx,
		apispec.SandboxID(s.ID),
		&params,
		"application/octet-stream",
		body,
	)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	if resp.JSON201 != nil {
		return nil, &APIError{Code: "unexpected_response", Message: "directory created instead of file"}
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Mkdir creates a directory.
func (s *Sandbox) Mkdir(ctx context.Context, path string, recursive bool) (*apispec.SuccessCreatedResponse, error) {
	params := apispec.PostApiV1SandboxesIdFilesParams{
		Path: apispec.FilePath(path),
	}
	mkdir := apispec.QueryMkdir(true)
	params.Mkdir = &mkdir
	if recursive {
		rec := apispec.QueryRecursive(true)
		params.Recursive = &rec
	}
	resp, err := s.client.api.PostApiV1SandboxesIdFilesWithBodyWithResponse(
		ctx,
		apispec.SandboxID(s.ID),
		&params,
		"application/octet-stream",
		bytes.NewReader(nil),
	)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 != nil {
		return resp.JSON201, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Delete deletes a file or directory.
func (s *Sandbox) DeleteFile(ctx context.Context, path string) (*apispec.SuccessDeletedResponse, error) {
	params := apispec.DeleteApiV1SandboxesIdFilesParams{
		Path: apispec.FilePath(path),
	}
	resp, err := s.client.api.DeleteApiV1SandboxesIdFilesWithResponse(ctx, apispec.SandboxID(s.ID), &params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Move moves a file or directory.
func (s *Sandbox) MoveFile(ctx context.Context, source, destination string) (*apispec.SuccessMovedResponse, error) {
	resp, err := s.client.api.PostApiV1SandboxesIdFilesMoveWithResponse(ctx, apispec.SandboxID(s.ID), apispec.MoveFileRequest{
		Source:      source,
		Destination: destination,
	})
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// ConnectWatch opens a WebSocket stream for file watch events.
func (s *Sandbox) ConnectWatchFile(ctx context.Context) (*websocket.Conn, *http.Response, error) {
	wsURL, err := s.client.websocketURL("/api/v1/sandboxes/" + s.ID + "/files/watch")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wsURL, nil)
	if err != nil {
		return nil, nil, err
	}
	if err := s.client.applyRequestEditors(ctx, req); err != nil {
		return nil, nil, err
	}

	return websocket.DefaultDialer.DialContext(ctx, wsURL, req.Header)
}

// Watch subscribes to file watch events and returns an unsubscribe handler.
func (s *Sandbox) WatchFiles(ctx context.Context, path string, recursive bool) (<-chan FileWatchResponse, <-chan error, func() error, error) {
	conn, _, err := s.ConnectWatchFile(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	subscribe := FileWatchSubscribeRequest{
		Action:    "subscribe",
		Path:      path,
		Recursive: recursive,
	}
	if err := conn.WriteJSON(subscribe); err != nil {
		_ = conn.Close()
		return nil, nil, nil, err
	}

	var resp FileWatchResponse
	if err := conn.ReadJSON(&resp); err != nil {
		_ = conn.Close()
		return nil, nil, nil, err
	}
	if resp.Type == "error" {
		_ = conn.Close()
		return nil, nil, nil, fmt.Errorf("watch subscribe failed: %s", resp.Error)
	}
	if resp.Type != "subscribed" || resp.WatchID == "" {
		_ = conn.Close()
		return nil, nil, nil, fmt.Errorf("unexpected watch response: %s", resp.Type)
	}

	unsubscribe := func() error {
		err := conn.WriteJSON(FileWatchUnsubscribeRequest{
			Action:  "unsubscribe",
			WatchID: resp.WatchID,
		})
		_ = conn.Close()
		return err
	}

	events := make(chan FileWatchResponse)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)
		for {
			var msg FileWatchResponse
			if err := conn.ReadJSON(&msg); err != nil {
				if ctx.Err() == nil {
					errs <- err
				}
				return
			}
			if msg.Type == "error" && msg.Error != "" {
				errs <- fmt.Errorf("watch error: %s", msg.Error)
				continue
			}
			events <- msg
		}
	}()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	return events, errs, unsubscribe, nil
}
