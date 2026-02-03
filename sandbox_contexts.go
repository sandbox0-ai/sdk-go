package sandbox0

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// ListContext returns all contexts for a sandbox.
func (s *Sandbox) ListContext(ctx context.Context) ([]apispec.ContextResponse, error) {
	resp, err := s.client.api.GetApiV1SandboxesIdContextsWithResponse(ctx, apispec.SandboxID(s.ID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil && resp.JSON200.Data.Contexts != nil {
		return *resp.JSON200.Data.Contexts, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// CreateContext creates a new context.
func (s *Sandbox) CreateContext(ctx context.Context, request apispec.CreateContextRequest) (*apispec.ContextResponse, error) {
	resp, err := s.client.api.PostApiV1SandboxesIdContextsWithResponse(ctx, apispec.SandboxID(s.ID), request)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 != nil && resp.JSON201.Data != nil {
		return resp.JSON201.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// GetContext returns a context by ID.
func (s *Sandbox) GetContext(ctx context.Context, contextID string) (*apispec.ContextResponse, error) {
	resp, err := s.client.api.GetApiV1SandboxesIdContextsCtxIdWithResponse(ctx, apispec.SandboxID(s.ID), apispec.ContextID(contextID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// DeleteContext deletes a context.
func (s *Sandbox) DeleteContext(ctx context.Context, contextID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := s.client.api.DeleteApiV1SandboxesIdContextsCtxIdWithResponse(ctx, apispec.SandboxID(s.ID), apispec.ContextID(contextID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// RestartContext restarts a context.
func (s *Sandbox) RestartContext(ctx context.Context, contextID string) (*apispec.ContextResponse, error) {
	resp, err := s.client.api.PostApiV1SandboxesIdContextsCtxIdRestartWithResponse(ctx, apispec.SandboxID(s.ID), apispec.ContextID(contextID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// ContextInput sends input to a context.
func (s *Sandbox) ContextInput(ctx context.Context, contextID string, input string) (*apispec.SuccessWrittenResponse, error) {
	resp, err := s.client.api.PostApiV1SandboxesIdContextsCtxIdInputWithResponse(ctx, apispec.SandboxID(s.ID), apispec.ContextID(contextID), apispec.ContextInputRequest{Data: input})
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// ContextExec sends input and waits for completion.
func (s *Sandbox) ContextExec(ctx context.Context, contextID string, input string) (*apispec.ContextExecResponse, error) {
	resp, err := s.client.api.PostApiV1SandboxesIdContextsCtxIdExecWithResponse(ctx, apispec.SandboxID(s.ID), apispec.ContextID(contextID), apispec.ContextInputRequest{Data: input})
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// ContextResize resizes a PTY context.
func (s *Sandbox) ContextResize(ctx context.Context, contextID string, rows, cols uint16) (*apispec.SuccessResizedResponse, error) {
	resp, err := s.client.api.PostApiV1SandboxesIdContextsCtxIdResizeWithResponse(ctx, apispec.SandboxID(s.ID), apispec.ContextID(contextID), apispec.ResizeContextRequest{
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

// ContextSignal sends a signal to a context.
func (s *Sandbox) ContextSignal(ctx context.Context, contextID, signal string) (*apispec.SuccessSignaledResponse, error) {
	resp, err := s.client.api.PostApiV1SandboxesIdContextsCtxIdSignalWithResponse(ctx, apispec.SandboxID(s.ID), apispec.ContextID(contextID), apispec.SignalContextRequest{Signal: signal})
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// ContextStats returns resource usage for a context.
func (s *Sandbox) ContextStats(ctx context.Context, contextID string) (*apispec.ContextStatsResponse, error) {
	resp, err := s.client.api.GetApiV1SandboxesIdContextsCtxIdStatsWithResponse(ctx, apispec.SandboxID(s.ID), apispec.ContextID(contextID))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil && resp.JSON200.Data != nil {
		return resp.JSON200.Data, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// ConnectWSContext opens a WebSocket stream for a context.
func (s *Sandbox) ConnectWSContext(ctx context.Context, contextID string) (*websocket.Conn, *http.Response, error) {
	wsURL, err := s.client.websocketURL("/api/v1/sandboxes/" + s.ID + "/contexts/" + contextID + "/ws")
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
