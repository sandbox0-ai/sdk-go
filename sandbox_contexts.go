package sandbox0

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// ListContext returns all contexts for a sandbox.
func (s *Sandbox) ListContext(ctx context.Context) ([]apispec.ContextResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDContextsGet(ctx, apispec.APIV1SandboxesIDContextsGetParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return data.Contexts, nil
}

// CreateContext creates a new context.
func (s *Sandbox) CreateContext(ctx context.Context, request apispec.CreateContextRequest) (*apispec.ContextResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDContextsPost(ctx, &request, apispec.APIV1SandboxesIDContextsPostParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}

// GetContext returns a context by ID.
func (s *Sandbox) GetContext(ctx context.Context, contextID string) (*apispec.ContextResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDContextsCtxIDGet(ctx, apispec.APIV1SandboxesIDContextsCtxIDGetParams{
		ID:    s.ID,
		CtxID: contextID,
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

// DeleteContext deletes a context.
func (s *Sandbox) DeleteContext(ctx context.Context, contextID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDContextsCtxIDDelete(ctx, apispec.APIV1SandboxesIDContextsCtxIDDeleteParams{
		ID:    s.ID,
		CtxID: contextID,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RestartContext restarts a context.
func (s *Sandbox) RestartContext(ctx context.Context, contextID string) (*apispec.ContextResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDContextsCtxIDRestartPost(ctx, apispec.APIV1SandboxesIDContextsCtxIDRestartPostParams{
		ID:    s.ID,
		CtxID: contextID,
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

// ContextInput sends input to a context.
func (s *Sandbox) ContextInput(ctx context.Context, contextID string, input string) (*apispec.SuccessWrittenResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDContextsCtxIDInputPost(ctx, &apispec.ContextInputRequest{Data: input}, apispec.APIV1SandboxesIDContextsCtxIDInputPostParams{
		ID:    s.ID,
		CtxID: contextID,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ContextExec sends input and waits for completion.
func (s *Sandbox) ContextExec(ctx context.Context, contextID string, input string) (*apispec.ContextExecResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDContextsCtxIDExecPost(ctx, &apispec.ContextInputRequest{Data: input}, apispec.APIV1SandboxesIDContextsCtxIDExecPostParams{
		ID:    s.ID,
		CtxID: contextID,
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

// ContextResize resizes a PTY context.
func (s *Sandbox) ContextResize(ctx context.Context, contextID string, rows, cols uint16) (*apispec.SuccessResizedResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDContextsCtxIDResizePost(ctx, &apispec.ResizeContextRequest{
		Rows: int32(rows),
		Cols: int32(cols),
	}, apispec.APIV1SandboxesIDContextsCtxIDResizePostParams{
		ID:    s.ID,
		CtxID: contextID,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ContextSignal sends a signal to a context.
func (s *Sandbox) ContextSignal(ctx context.Context, contextID, signal string) (*apispec.SuccessSignaledResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDContextsCtxIDSignalPost(ctx, &apispec.SignalContextRequest{Signal: signal}, apispec.APIV1SandboxesIDContextsCtxIDSignalPostParams{
		ID:    s.ID,
		CtxID: contextID,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ContextStats returns resource usage for a context.
func (s *Sandbox) ContextStats(ctx context.Context, contextID string) (*apispec.ContextStatsResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDContextsCtxIDStatsGet(ctx, apispec.APIV1SandboxesIDContextsCtxIDStatsGetParams{
		ID:    s.ID,
		CtxID: contextID,
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

	var lastErr error
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		conn, resp, dialErr := websocket.DefaultDialer.DialContext(ctx, wsURL, req.Header)
		if dialErr == nil {
			return conn, resp, nil
		}
		lastErr = dialErr
		if ctx.Err() != nil || !isRetryableWSDialError(dialErr) || attempt == maxAttempts {
			return nil, resp, dialErr
		}

		delay := time.Duration(attempt) * 150 * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, nil, lastErr
}

func isRetryableWSDialError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "connection reset by peer") ||
		strings.Contains(lower, "broken pipe") ||
		strings.Contains(lower, "unexpected eof")
}
