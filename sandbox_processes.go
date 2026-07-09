package sandbox0

import (
	"bufio"
	"bytes"
	"context"
	stdjson "encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-faster/jx"
	"github.com/google/uuid"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// ProcessEventWatchOptions configures process event replay.
type ProcessEventWatchOptions struct {
	// Cursor is the last observed event sequence. The stream replays events after it.
	Cursor *int64
}

// ProcessEventStream reads replayed and live process events from SSE.
type ProcessEventStream struct {
	io.ReadCloser
	scanner *bufio.Scanner
	buffer  bytes.Buffer
}

// Recv reads the next process event from the stream.
func (s *ProcessEventStream) Recv() (*apispec.ProcessEvent, error) {
	if s == nil || s.scanner == nil {
		return nil, io.EOF
	}
	for s.scanner.Scan() {
		line := s.scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			if s.buffer.Len() == 0 {
				continue
			}
			event, err := decodeProcessSSEData(s.buffer.Bytes())
			s.buffer.Reset()
			return event, err
		}
		if bytes.HasPrefix(line, []byte("data:")) {
			if s.buffer.Len() > 0 {
				s.buffer.WriteByte('\n')
			}
			s.buffer.Write(bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:"))))
		}
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	if s.buffer.Len() > 0 {
		event, err := decodeProcessSSEData(s.buffer.Bytes())
		s.buffer.Reset()
		return event, err
	}
	return nil, io.EOF
}

// NewProcessPayload converts a plain map into the generated OpenAPI payload type.
func NewProcessPayload(values map[string]any) (apispec.ProcessInputEventPayload, error) {
	payload := make(apispec.ProcessInputEventPayload, len(values))
	for key, value := range values {
		raw, err := stdjson.Marshal(value)
		if err != nil {
			return nil, err
		}
		payload[key] = jx.Raw(raw)
	}
	return payload, nil
}

// NewProcessInputEvent builds an idempotent process input event.
func NewProcessInputEvent(eventID, channel string, eventType apispec.ProcessEventType, payload map[string]any) (apispec.ProcessInputEvent, error) {
	if strings.TrimSpace(eventID) == "" {
		eventID = uuid.NewString()
	}
	event := apispec.ProcessInputEvent{
		EventID: eventID,
		Channel: channel,
		Type:    eventType,
	}
	if payload != nil {
		converted, err := NewProcessPayload(payload)
		if err != nil {
			return apispec.ProcessInputEvent{}, err
		}
		event.Payload = apispec.NewOptProcessInputEventPayload(converted)
	}
	return event, nil
}

// NewProcessStdinEvent builds a stdin.write event with a data payload.
func NewProcessStdinEvent(eventID, channel, data string) (apispec.ProcessInputEvent, error) {
	return NewProcessInputEvent(eventID, channel, apispec.ProcessEventTypeStdinWrite, map[string]any{"data": data})
}

// NewProcessPTYInputEvent builds a pty.input event with a data payload.
func NewProcessPTYInputEvent(eventID, channel, data string) (apispec.ProcessInputEvent, error) {
	return NewProcessInputEvent(eventID, channel, apispec.ProcessEventTypePtyInput, map[string]any{"data": data})
}

// StdioProcessSpec builds a common stdio process spec.
func StdioProcessSpec(command []string) apispec.ProcessSpec {
	return apispec.ProcessSpec{
		Command: command,
		Channels: []apispec.ProcessChannelSpec{
			{
				Name:    "stdio",
				Kind:    apispec.ProcessChannelKindStdio,
				Framing: apispec.NewOptProcessChannelFraming(apispec.ProcessChannelFramingLine),
				Stdin:   apispec.NewOptBool(true),
				Stdout:  apispec.NewOptBool(true),
				Stderr:  apispec.NewOptBool(true),
			},
		},
	}
}

// CreateProcess creates a broker-owned process session.
func (s *Sandbox) CreateProcess(ctx context.Context, spec apispec.ProcessSpec) (*apispec.ProcessSession, error) {
	resp, err := s.client.api.APIV1SandboxesIDProcessesPost(ctx, &spec, apispec.APIV1SandboxesIDProcessesPostParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	process, ok := data.Process.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &process, nil
}

// ListProcesses returns process sessions in the sandbox.
func (s *Sandbox) ListProcesses(ctx context.Context) ([]apispec.ProcessSession, error) {
	resp, err := s.client.api.APIV1SandboxesIDProcessesGet(ctx, apispec.APIV1SandboxesIDProcessesGetParams{ID: s.ID})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return data.Processes, nil
}

// GetProcess returns a process session by ID.
func (s *Sandbox) GetProcess(ctx context.Context, processID string) (*apispec.ProcessSession, error) {
	resp, err := s.client.api.APIV1SandboxesIDProcessesProcessIDGet(ctx, apispec.APIV1SandboxesIDProcessesProcessIDGetParams{
		ID:        s.ID,
		ProcessID: processID,
	})
	if err != nil {
		return nil, err
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	process, ok := data.Process.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &process, nil
}

// DeleteProcess stops and removes a process session.
func (s *Sandbox) DeleteProcess(ctx context.Context, processID string) (*apispec.SuccessDeletedResponse, error) {
	return s.client.api.APIV1SandboxesIDProcessesProcessIDDelete(ctx, apispec.APIV1SandboxesIDProcessesProcessIDDeleteParams{
		ID:        s.ID,
		ProcessID: processID,
	})
}

// SendProcessEvent sends an idempotent input event to a process channel.
func (s *Sandbox) SendProcessEvent(ctx context.Context, processID string, event apispec.ProcessInputEvent) (*apispec.ProcessEvent, error) {
	resp, err := s.client.api.APIV1SandboxesIDProcessesProcessIDEventsPost(ctx, &event, apispec.APIV1SandboxesIDProcessesProcessIDEventsPostParams{
		ID:        s.ID,
		ProcessID: processID,
	})
	if err != nil {
		return nil, err
	}
	if success, ok := resp.(*apispec.SuccessProcessEventResponse); ok {
		data, ok := success.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(resp)
		}
		accepted, ok := data.Event.Get()
		if !ok {
			return nil, unexpectedResponseError(resp)
		}
		return &accepted, nil
	}
	return nil, apiErrorFromResponse(resp)
}

// SendProcessInput sends stdin.write data to a process channel.
func (s *Sandbox) SendProcessInput(ctx context.Context, processID, channel, data string) (*apispec.ProcessEvent, error) {
	event, err := NewProcessStdinEvent("", channel, data)
	if err != nil {
		return nil, err
	}
	return s.SendProcessEvent(ctx, processID, event)
}

// SignalProcess sends a signal to a process session.
func (s *Sandbox) SignalProcess(ctx context.Context, processID, signal string) (*apispec.SuccessSignaledResponse, error) {
	return s.client.api.APIV1SandboxesIDProcessesProcessIDSignalPost(ctx, &apispec.SignalContextRequest{Signal: signal}, apispec.APIV1SandboxesIDProcessesProcessIDSignalPostParams{
		ID:        s.ID,
		ProcessID: processID,
	})
}

// ResizeProcessPTY resizes a PTY channel.
func (s *Sandbox) ResizeProcessPTY(ctx context.Context, processID, channel string, rows, cols uint16) (*apispec.SuccessResizedResponse, error) {
	return s.client.api.APIV1SandboxesIDProcessesProcessIDChannelsChannelPtySizePut(ctx, &apispec.ResizeContextRequest{
		Rows: int32(rows),
		Cols: int32(cols),
	}, apispec.APIV1SandboxesIDProcessesProcessIDChannelsChannelPtySizePutParams{
		ID:        s.ID,
		ProcessID: processID,
		Channel:   channel,
	})
}

// WatchProcessEvents opens an SSE stream for process events.
func (s *Sandbox) WatchProcessEvents(ctx context.Context, processID string, opts *ProcessEventWatchOptions) (*ProcessEventStream, error) {
	values := make(url.Values)
	if opts != nil && opts.Cursor != nil {
		values.Set("cursor", strconv.FormatInt(*opts.Cursor, 10))
	}
	reqURL, err := s.client.processEventsURL(s.ID, processID, values)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	if err := s.client.applyRequestEditors(ctx, req); err != nil {
		return nil, err
	}
	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes+1))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		truncated := false
		if len(body) > maxErrorBodyBytes {
			body = body[:maxErrorBodyBytes]
			truncated = true
		}
		return nil, apiErrorFromHTTPResponse(resp, body, truncated)
	}
	if contentType := resp.Header.Get("Content-Type"); contentType != "" && !strings.HasPrefix(strings.ToLower(contentType), "text/event-stream") {
		_ = resp.Body.Close()
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Code:       "unexpected_response",
			Message:    fmt.Sprintf("unexpected process event stream content type: %s", contentType),
		}
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	return &ProcessEventStream{ReadCloser: resp.Body, scanner: scanner}, nil
}

func (c *Client) processEventsURL(sandboxID, processID string, values url.Values) (string, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	baseURL.Path = strings.TrimSuffix(baseURL.Path, "/") + "/api/v1/sandboxes/" + url.PathEscape(sandboxID) + "/processes/" + url.PathEscape(processID) + "/events"
	baseURL.RawQuery = values.Encode()
	return baseURL.String(), nil
}

func decodeProcessSSEData(data []byte) (*apispec.ProcessEvent, error) {
	var event apispec.ProcessEvent
	if err := stdjson.Unmarshal(bytes.TrimSpace(data), &event); err != nil {
		return nil, err
	}
	return &event, nil
}
