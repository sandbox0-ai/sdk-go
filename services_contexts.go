package sandbox0

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// SandboxContextService provides context APIs scoped to a sandbox.
type SandboxContextService struct {
	sandbox *Sandbox
}

// List returns all contexts for a sandbox.
func (s *SandboxContextService) List(ctx context.Context) ([]apispec.ContextResponse, error) {
	resp, err := s.sandbox.client.api.GetApiV1SandboxesIdContextsWithResponse(ctx, apispec.SandboxID(s.sandbox.ID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil && resp.JSON200.Data.Contexts != nil {
		return *resp.JSON200.Data.Contexts, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Create creates a new context.
func (s *SandboxContextService) Create(ctx context.Context, request apispec.CreateContextRequest) (*apispec.ContextResponse, error) {
	resp, err := s.sandbox.client.api.PostApiV1SandboxesIdContextsWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), request)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 != nil && resp.JSON201.Data != nil {
		return resp.JSON201.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Get returns a context by ID.
func (s *SandboxContextService) Get(ctx context.Context, contextID string) (*apispec.ContextResponse, error) {
	resp, err := s.sandbox.client.api.GetApiV1SandboxesIdContextsCtxIdWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), apispec.ContextID(contextID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Delete deletes a context.
func (s *SandboxContextService) Delete(ctx context.Context, contextID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := s.sandbox.client.api.DeleteApiV1SandboxesIdContextsCtxIdWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), apispec.ContextID(contextID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Restart restarts a context.
func (s *SandboxContextService) Restart(ctx context.Context, contextID string) (*apispec.ContextResponse, error) {
	resp, err := s.sandbox.client.api.PostApiV1SandboxesIdContextsCtxIdRestartWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), apispec.ContextID(contextID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Input sends input to a context.
func (s *SandboxContextService) Input(ctx context.Context, contextID string, input string) (*apispec.SuccessWrittenResponse, error) {
	resp, err := s.sandbox.client.api.PostApiV1SandboxesIdContextsCtxIdInputWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), apispec.ContextID(contextID), apispec.ContextInputRequest{Data: input})
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Exec sends input and waits for completion.
func (s *SandboxContextService) Exec(ctx context.Context, contextID string, input string) (*apispec.ContextExecResponse, error) {
	resp, err := s.sandbox.client.api.PostApiV1SandboxesIdContextsCtxIdExecWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), apispec.ContextID(contextID), apispec.ContextInputRequest{Data: input})
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Resize resizes a PTY context.
func (s *SandboxContextService) Resize(ctx context.Context, contextID string, rows, cols uint16) (*apispec.SuccessResizedResponse, error) {
	resp, err := s.sandbox.client.api.PostApiV1SandboxesIdContextsCtxIdResizeWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), apispec.ContextID(contextID), apispec.ResizeContextRequest{
		Rows: int32(rows),
		Cols: int32(cols),
	})
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Signal sends a signal to a context.
func (s *SandboxContextService) Signal(ctx context.Context, contextID, signal string) (*apispec.SuccessSignaledResponse, error) {
	resp, err := s.sandbox.client.api.PostApiV1SandboxesIdContextsCtxIdSignalWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), apispec.ContextID(contextID), apispec.SignalContextRequest{Signal: signal})
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Stats returns resource usage for a context.
func (s *SandboxContextService) Stats(ctx context.Context, contextID string) (*apispec.ContextStatsResponse, error) {
	resp, err := s.sandbox.client.api.GetApiV1SandboxesIdContextsCtxIdStatsWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), apispec.ContextID(contextID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// ConnectWS opens a WebSocket stream for a context.
func (s *SandboxContextService) ConnectWS(ctx context.Context, contextID string) (*websocket.Conn, *http.Response, error) {
	wsURL, err := s.sandbox.client.websocketURL("/api/v1/sandboxes/" + s.sandbox.ID + "/contexts/" + contextID + "/ws")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wsURL, nil)
	if err != nil {
		return nil, nil, err
	}
	if err := s.sandbox.client.applyRequestEditors(ctx, req); err != nil {
		return nil, nil, err
	}

	return websocket.DefaultDialer.DialContext(ctx, wsURL, req.Header)
}

func (c *Client) websocketURL(path string) (string, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(baseURL.Scheme) {
	case "https":
		baseURL.Scheme = "wss"
	case "http":
		baseURL.Scheme = "ws"
	}
	baseURL.Path = strings.TrimSuffix(baseURL.Path, "/") + path
	return baseURL.String(), nil
}
