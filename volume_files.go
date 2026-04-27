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

// ReadVolumeFile reads a file from a volume and returns raw bytes.
func (c *Client) ReadVolumeFile(ctx context.Context, volumeID, path string) ([]byte, error) {
	params := apispec.APIV1SandboxvolumesIDFilesGetParams{
		ID:   volumeID,
		Path: path,
	}
	resp, err := c.api.APIV1SandboxvolumesIDFilesGet(ctx, params)
	if err != nil {
		return nil, err
	}
	return decodeVolumeFileGetResponse(resp)
}

// StatVolumeFile retrieves volume file metadata.
func (c *Client) StatVolumeFile(ctx context.Context, volumeID, path string) (*apispec.FileInfo, error) {
	resp, err := c.api.APIV1SandboxvolumesIDFilesStatGet(ctx, apispec.APIV1SandboxvolumesIDFilesStatGetParams{
		ID:   volumeID,
		Path: path,
	})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}

// ListVolumeFiles returns directory entries inside a volume path.
func (c *Client) ListVolumeFiles(ctx context.Context, volumeID, path string) ([]apispec.FileInfo, error) {
	resp, err := c.api.APIV1SandboxvolumesIDFilesListGet(ctx, apispec.APIV1SandboxvolumesIDFilesListGetParams{
		ID:   volumeID,
		Path: path,
	})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return data.Entries, nil
}

// WriteVolumeFile writes file content into a volume path.
func (c *Client) WriteVolumeFile(ctx context.Context, volumeID, path string, data []byte) (*apispec.SuccessWrittenResponse, error) {
	resp, err := c.api.APIV1SandboxvolumesIDFilesPost(ctx, apispec.APIV1SandboxvolumesIDFilesPostReq{
		Data: bytes.NewReader(data),
	}, apispec.APIV1SandboxvolumesIDFilesPostParams{
		ID:   volumeID,
		Path: path,
	})
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

// MkdirVolumeFile creates a directory inside a volume.
func (c *Client) MkdirVolumeFile(ctx context.Context, volumeID, path string, recursive bool) (*apispec.SuccessCreatedResponse, error) {
	params := apispec.APIV1SandboxvolumesIDFilesPostParams{
		ID:   volumeID,
		Path: path,
	}
	params.Mkdir = apispec.NewOptBool(true)
	if recursive {
		params.Recursive = apispec.NewOptBool(true)
	}

	resp, err := c.api.APIV1SandboxvolumesIDFilesPost(ctx, apispec.APIV1SandboxvolumesIDFilesPostReq{
		Data: bytes.NewReader(nil),
	}, params)
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

// DeleteVolumeFile deletes a file or directory from a volume.
func (c *Client) DeleteVolumeFile(ctx context.Context, volumeID, path string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := c.api.APIV1SandboxvolumesIDFilesDelete(ctx, apispec.APIV1SandboxvolumesIDFilesDeleteParams{
		ID:   volumeID,
		Path: path,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// MoveVolumeFile moves a file or directory within a volume.
func (c *Client) MoveVolumeFile(ctx context.Context, volumeID, source, destination string) (*apispec.SuccessMovedResponse, error) {
	resp, err := c.api.APIV1SandboxvolumesIDFilesMovePost(ctx, &apispec.MoveFileRequest{
		Source:      source,
		Destination: destination,
	}, apispec.APIV1SandboxvolumesIDFilesMovePostParams{ID: volumeID})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// CloneVolumeFiles clones files from source volumes into a target volume.
func (c *Client) CloneVolumeFiles(ctx context.Context, volumeID string, request apispec.CloneVolumeFilesRequest) ([]apispec.CloneVolumeFileResult, error) {
	resp, err := c.api.APIV1SandboxvolumesIDFilesClonePost(ctx, &request, apispec.APIV1SandboxvolumesIDFilesClonePostParams{ID: volumeID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessCloneVolumeFilesResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(resp)
		}
		return data.Entries, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ConnectWatchVolumeFile opens a WebSocket stream for volume file watch events.
func (c *Client) ConnectWatchVolumeFile(ctx context.Context, volumeID string) (*websocket.Conn, *http.Response, error) {
	wsURL, err := c.websocketURL("/api/v1/sandboxvolumes/" + volumeID + "/files/watch")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wsURL, nil)
	if err != nil {
		return nil, nil, err
	}
	if err := c.applyRequestEditors(ctx, req); err != nil {
		return nil, nil, err
	}

	return websocket.DefaultDialer.DialContext(ctx, wsURL, req.Header)
}

// WatchVolumeFiles subscribes to volume file watch events.
func (c *Client) WatchVolumeFiles(ctx context.Context, volumeID, path string, recursive bool) (<-chan FileWatchResponse, <-chan error, func() error, error) {
	conn, _, err := c.ConnectWatchVolumeFile(ctx, volumeID)
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

	return events, errs, unsubscribe, nil
}

func decodeVolumeFileGetResponse(resp apispec.APIV1SandboxvolumesIDFilesGetRes) ([]byte, error) {
	switch response := resp.(type) {
	case *apispec.APIV1SandboxvolumesIDFilesGetOKApplicationOctetStream:
		return io.ReadAll(response)
	case *apispec.APIV1SandboxvolumesIDFilesGetOKApplicationJSON:
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
