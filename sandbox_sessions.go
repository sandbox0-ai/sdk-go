package sandbox0

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// CreateSessionOptions configures execution session creation.
type CreateSessionOptions struct {
	// IdempotencyKey lets callers safely retry a create request.
	IdempotencyKey string
}

// SessionEventOptions selects a page from a session event journal.
type SessionEventOptions struct {
	After int64
	Limit int
}

// SessionEventStreamOptions selects where an SSE attachment starts replaying.
type SessionEventStreamOptions struct {
	After       int64
	LastEventID string
}

// SessionWebSocketOptions selects where a WebSocket attachment starts replaying.
type SessionWebSocketOptions struct {
	After int64
}

// ListSessions returns all durable execution sessions in the sandbox.
func (s *Sandbox) ListSessions(ctx context.Context) ([]apispec.ExecutionSession, error) {
	resp, err := s.client.api.APIV1SandboxesIDSessionsGet(ctx, apispec.APIV1SandboxesIDSessionsGetParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessExecutionSessionListResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return data.Sessions, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// CreateSession creates a durable execution session.
func (s *Sandbox) CreateSession(ctx context.Context, spec apispec.ExecutionSessionSpec, opts *CreateSessionOptions) (*apispec.ExecutionSession, error) {
	params := apispec.APIV1SandboxesIDSessionsPostParams{ID: s.ID}
	if opts != nil && strings.TrimSpace(opts.IdempotencyKey) != "" {
		params.IdempotencyKey = apispec.NewOptString(opts.IdempotencyKey)
	}
	resp, err := s.client.api.APIV1SandboxesIDSessionsPost(ctx, &spec, params)
	if err != nil {
		return nil, err
	}
	switch value := resp.(type) {
	case *apispec.APIV1SandboxesIDSessionsPostCreated:
		return executionSessionData((*apispec.SuccessExecutionSessionResponse)(value))
	case *apispec.APIV1SandboxesIDSessionsPostOK:
		return executionSessionData((*apispec.SuccessExecutionSessionResponse)(value))
	default:
		return nil, apiErrorFromResponse(resp)
	}
}

// GetSession returns a durable execution session by ID.
func (s *Sandbox) GetSession(ctx context.Context, sessionID string) (*apispec.ExecutionSession, error) {
	resp, err := s.client.api.APIV1SandboxesIDSessionsSessionIDGet(ctx, apispec.APIV1SandboxesIDSessionsSessionIDGetParams{
		ID: s.ID, SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}
	return executionSessionData(resp)
}

// UpdateSession replaces a session specification.
func (s *Sandbox) UpdateSession(ctx context.Context, sessionID string, spec apispec.ExecutionSessionSpec) (*apispec.ExecutionSession, error) {
	resp, err := s.client.api.APIV1SandboxesIDSessionsSessionIDPut(ctx, &spec, apispec.APIV1SandboxesIDSessionsSessionIDPutParams{
		ID: s.ID, SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}
	return executionSessionData(resp)
}

// DeleteSession stops and deletes a durable execution session and its journal.
func (s *Sandbox) DeleteSession(ctx context.Context, sessionID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDSessionsSessionIDDelete(ctx, apispec.APIV1SandboxesIDSessionsSessionIDDeleteParams{
		ID: s.ID, SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessDeletedResponse:
		return response, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// SetSessionDesiredState starts or stops a session without deleting its identity or journal.
func (s *Sandbox) SetSessionDesiredState(ctx context.Context, sessionID string, state apispec.ExecutionSessionDesiredState) (*apispec.ExecutionSession, error) {
	resp, err := s.client.api.APIV1SandboxesIDSessionsSessionIDDesiredStatePut(ctx, &apispec.ExecutionSessionDesiredStateRequest{State: state}, apispec.APIV1SandboxesIDSessionsSessionIDDesiredStatePutParams{
		ID: s.ID, SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}
	return executionSessionData(resp)
}

// CreateSessionAttempt starts a new attempt. Set replaceCurrent to stop a running attempt first.
func (s *Sandbox) CreateSessionAttempt(ctx context.Context, sessionID string, replaceCurrent bool) (*apispec.ExecutionSession, error) {
	request := apispec.NewOptCreateExecutionSessionAttemptRequest(apispec.CreateExecutionSessionAttemptRequest{
		ReplaceCurrent: apispec.NewOptBool(replaceCurrent),
	})
	resp, err := s.client.api.APIV1SandboxesIDSessionsSessionIDAttemptsPost(ctx, request, apispec.APIV1SandboxesIDSessionsSessionIDAttemptsPostParams{
		ID: s.ID, SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}
	return executionSessionData(resp)
}

// WriteSessionInput writes bytes, an explicit EOF, or both to the current attempt.
func (s *Sandbox) WriteSessionInput(ctx context.Context, sessionID string, request apispec.ExecutionSessionInputRequest) (*apispec.ExecutionSessionInputResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDSessionsSessionIDInputsPost(ctx, &request, apispec.APIV1SandboxesIDSessionsSessionIDInputsPostParams{
		ID: s.ID, SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessExecutionSessionInputResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// SendSessionSignal sends a signal to the current attempt.
func (s *Sandbox) SendSessionSignal(ctx context.Context, sessionID string, request apispec.ExecutionSessionSignalRequest) (*apispec.SuccessAcceptedResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDSessionsSessionIDSignalsPost(ctx, &request, apispec.APIV1SandboxesIDSessionsSessionIDSignalsPostParams{
		ID: s.ID, SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessAcceptedResponse:
		return response, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ResizeSessionTerminal resizes a PTY session's current attempt.
func (s *Sandbox) ResizeSessionTerminal(ctx context.Context, sessionID string, request apispec.ExecutionSessionTerminalResizeRequest) (*apispec.SuccessResizedResponse, error) {
	resp, err := s.client.api.APIV1SandboxesIDSessionsSessionIDTerminalPut(ctx, &request, apispec.APIV1SandboxesIDSessionsSessionIDTerminalPutParams{
		ID: s.ID, SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessResizedResponse:
		return response, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ListSessionEvents returns a replayable page from the durable event journal.
func (s *Sandbox) ListSessionEvents(ctx context.Context, sessionID string, opts *SessionEventOptions) (*apispec.ExecutionSessionEventPage, error) {
	params := apispec.APIV1SandboxesIDSessionsSessionIDEventsGetParams{ID: s.ID, SessionID: sessionID}
	if opts != nil {
		if opts.After > 0 {
			params.After = apispec.NewOptInt64(opts.After)
		}
		if opts.Limit > 0 {
			params.Limit = apispec.NewOptInt(opts.Limit)
		}
	}
	resp, err := s.client.api.APIV1SandboxesIDSessionsSessionIDEventsGet(ctx, params)
	if err != nil {
		return nil, err
	}
	value, ok := resp.(*apispec.SuccessExecutionSessionEventPageResponse)
	if !ok {
		return nil, apiErrorFromResponse(resp)
	}
	data, ok := value.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data, nil
}

// SessionEventStream is a resumable SSE attachment to a session journal.
// Closing it detaches the client; it does not stop the session or close stdin.
type SessionEventStream struct {
	io.ReadCloser
	scanner *bufio.Scanner
}

// Recv waits for and decodes the next session event.
func (s *SessionEventStream) Recv() (*apispec.ExecutionSessionEvent, error) {
	if s == nil || s.scanner == nil {
		return nil, io.EOF
	}
	var data strings.Builder
	for s.scanner.Scan() {
		line := s.scanner.Text()
		if line == "" {
			if data.Len() == 0 {
				continue
			}
			var event apispec.ExecutionSessionEvent
			if err := json.Unmarshal([]byte(data.String()), &event); err != nil {
				return nil, err
			}
			return &event, nil
		}
		if strings.HasPrefix(line, "data:") {
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// WatchSessionEvents opens a resumable SSE attachment to a session journal.
func (s *Sandbox) WatchSessionEvents(ctx context.Context, sessionID string, opts *SessionEventStreamOptions) (*SessionEventStream, error) {
	values := url.Values{}
	if opts != nil && opts.After > 0 {
		values.Set("after", strconv.FormatInt(opts.After, 10))
	}
	reqURL, err := s.sessionURL(sessionID, "/events/stream", values)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	if opts != nil && strings.TrimSpace(opts.LastEventID) != "" {
		req.Header.Set("Last-Event-ID", opts.LastEventID)
	}
	if err := s.client.applyRequestEditors(ctx, req); err != nil {
		return nil, err
	}
	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, sessionStreamHTTPError(resp)
	}
	if contentType := strings.ToLower(resp.Header.Get("Content-Type")); contentType != "" && !strings.HasPrefix(contentType, "text/event-stream") {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("unexpected session event stream content type: %s", contentType)
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	return &SessionEventStream{ReadCloser: resp.Body, scanner: scanner}, nil
}

// SessionWebSocketRequest is a typed input, signal, or terminal resize message.
type SessionWebSocketRequest struct {
	Type              string `json:"type"`
	RequestID         string `json:"request_id,omitempty"`
	InputID           string `json:"input_id,omitempty"`
	ExpectedAttemptID string `json:"expected_attempt_id,omitempty"`
	DataBase64        string `json:"data_base64,omitempty"`
	EOF               bool   `json:"eof,omitempty"`
	Signal            string `json:"signal,omitempty"`
	Rows              uint16 `json:"rows,omitempty"`
	Cols              uint16 `json:"cols,omitempty"`
}

// SessionWebSocketMessage is an acknowledgement, event, or operation error.
type SessionWebSocketMessage struct {
	Type      string                         `json:"type"`
	RequestID string                         `json:"request_id,omitempty"`
	Event     *apispec.ExecutionSessionEvent `json:"event,omitempty"`
	Error     string                         `json:"error,omitempty"`
}

// SessionConnection is a duplex WebSocket attachment. Closing the connection
// does not stop the session or close its stdin.
type SessionConnection struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

// Send writes a typed operation to the attachment.
func (c *SessionConnection) Send(request SessionWebSocketRequest) error {
	if c == nil || c.conn == nil {
		return errors.New("session connection is closed")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(request)
}

// Recv reads the next acknowledgement or event.
func (c *SessionConnection) Recv() (*SessionWebSocketMessage, error) {
	if c == nil || c.conn == nil {
		return nil, io.EOF
	}
	var message SessionWebSocketMessage
	if err := c.conn.ReadJSON(&message); err != nil {
		return nil, err
	}
	return &message, nil
}

// WriteControl sends a WebSocket control frame.
func (c *SessionConnection) WriteControl(messageType int, data []byte, deadline time.Time) error {
	if c == nil || c.conn == nil {
		return errors.New("session connection is closed")
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteControl(messageType, data, deadline)
}

// Close detaches this client without changing session state.
func (c *SessionConnection) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// ConnectSession opens a duplex WebSocket attachment with optional event replay.
func (s *Sandbox) ConnectSession(ctx context.Context, sessionID string, opts *SessionWebSocketOptions) (*SessionConnection, *http.Response, error) {
	values := url.Values{}
	if opts != nil && opts.After > 0 {
		values.Set("after", strconv.FormatInt(opts.After, 10))
	}
	httpURL, err := s.sessionURL(sessionID, "/ws", values)
	if err != nil {
		return nil, nil, err
	}
	parsed, err := url.Parse(httpURL)
	if err != nil {
		return nil, nil, err
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	}
	wsURL := parsed.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wsURL, nil)
	if err != nil {
		return nil, nil, err
	}
	if err := s.client.applyRequestEditors(ctx, req); err != nil {
		return nil, nil, err
	}
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, req.URL.String(), req.Header)
	if err != nil {
		return nil, resp, err
	}
	return &SessionConnection{conn: conn}, resp, nil
}

func (s *Sandbox) sessionURL(sessionID, suffix string, values url.Values) (string, error) {
	baseURL, err := url.Parse(s.client.baseURL)
	if err != nil {
		return "", err
	}
	baseURL.Path = strings.TrimSuffix(baseURL.Path, "/") + "/api/v1/sandboxes/" + url.PathEscape(s.ID) + "/sessions/" + url.PathEscape(sessionID) + suffix
	baseURL.RawQuery = values.Encode()
	return baseURL.String(), nil
}

func executionSessionData(resp any) (*apispec.ExecutionSession, error) {
	response, ok := resp.(*apispec.SuccessExecutionSessionResponse)
	if !ok {
		return nil, apiErrorFromResponse(resp)
	}
	data, ok := response.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(response)
	}
	return &data, nil
}

func sessionStreamHTTPError(resp *http.Response) error {
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes+1))
	_ = resp.Body.Close()
	if readErr != nil {
		return readErr
	}
	truncated := len(body) > maxErrorBodyBytes
	if truncated {
		body = body[:maxErrorBodyBytes]
	}
	return apiErrorFromHTTPResponse(resp, body, truncated)
}
