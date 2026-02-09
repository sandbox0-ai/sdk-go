package sandbox0

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
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

// ReadFile reads a file and returns raw bytes.
func (s *Sandbox) ReadFile(ctx context.Context, path string) ([]byte, error) {
	params := apispec.APIV1SandboxesIDFilesGetParams{
		ID:   s.ID,
		Path: path,
	}
	resp, err := s.client.api.APIV1SandboxesIDFilesGet(ctx, params)
	if err != nil {
		return nil, err
	}
	return decodeFileGetResponse(resp)
}

func decodeFileGetResponse(resp apispec.APIV1SandboxesIDFilesGetRes) ([]byte, error) {
	switch response := resp.(type) {
	case *apispec.APIV1SandboxesIDFilesGetOKApplicationOctetStream:
		return io.ReadAll(response)
	case *apispec.APIV1SandboxesIDFilesGetOKApplicationJSON:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(resp)
		}
		content, ok := data.Content.Get()
		if !ok {
			return nil, unexpectedResponseError(resp)
		}
		if encoding, ok := data.Encoding.Get(); ok && encoding != apispec.FileContentResponseEncodingBase64 {
			return nil, &APIError{
				Code:    "unexpected_response",
				Message: fmt.Sprintf("unsupported file encoding: %s", encoding),
			}
		}
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return nil, err
		}
		return decoded, nil
	default:
		return nil, unexpectedResponseError(resp)
	}
}

// StatFile retrieves file metadata.
func (s *Sandbox) StatFile(ctx context.Context, path string) (*apispec.FileInfo, error) {
	params := apispec.APIV1SandboxesIDFilesStatGetParams{
		ID:   s.ID,
		Path: path,
	}
	resp, err := s.client.api.APIV1SandboxesIDFilesStatGet(ctx, params)
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}

// ListFiles returns directory entries.
func (s *Sandbox) ListFiles(ctx context.Context, path string) ([]apispec.FileInfo, error) {
	params := apispec.APIV1SandboxesIDFilesListGetParams{
		ID:   s.ID,
		Path: path,
	}
	resp, err := s.client.api.APIV1SandboxesIDFilesListGet(ctx, params)
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return data.Entries, nil
}

// WriteFile writes a file.
func (s *Sandbox) WriteFile(ctx context.Context, path string, data []byte) (*apispec.SuccessWrittenResponse, error) {
	body := bytes.NewReader(data)
	params := apispec.APIV1SandboxesIDFilesPostParams{
		ID:   s.ID,
		Path: path,
	}
	resp, err := s.client.api.APIV1SandboxesIDFilesPost(ctx, apispec.APIV1SandboxesIDFilesPostReq{Data: body}, params)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessWrittenResponse:
		return response, nil
	case *apispec.SuccessCreatedResponse:
		return nil, &APIError{Code: "unexpected_response", Message: "directory created instead of file"}
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// Mkdir creates a directory.
func (s *Sandbox) Mkdir(ctx context.Context, path string, recursive bool) (*apispec.SuccessCreatedResponse, error) {
	params := apispec.APIV1SandboxesIDFilesPostParams{
		ID:   s.ID,
		Path: path,
	}
	params.Mkdir = apispec.NewOptBool(true)
	if recursive {
		params.Recursive = apispec.NewOptBool(true)
	}
	resp, err := s.client.api.APIV1SandboxesIDFilesPost(ctx, apispec.APIV1SandboxesIDFilesPostReq{Data: bytes.NewReader(nil)}, params)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessCreatedResponse:
		return response, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteFile deletes a file or directory.
func (s *Sandbox) DeleteFile(ctx context.Context, path string) (*apispec.SuccessDeletedResponse, error) {
	params := apispec.APIV1SandboxesIDFilesDeleteParams{
		ID:   s.ID,
		Path: path,
	}
	resp, err := s.client.api.APIV1SandboxesIDFilesDelete(ctx, params)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// MoveFile moves a file or directory.
func (s *Sandbox) MoveFile(ctx context.Context, source, destination string) (*apispec.SuccessMovedResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDFilesMovePost(ctx, &apispec.MoveFileRequest{
		Source:      source,
		Destination: destination,
	}, apispec.APIV1SandboxesIDFilesMovePostParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ConnectWatchFile opens a WebSocket stream for file watch events.
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

// WatchFiles subscribes to file watch events and returns an unsubscribe handler.
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
