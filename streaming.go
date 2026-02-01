package sandbox0

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// StreamInputType describes the WebSocket control message type.
type StreamInputType string

const (
	StreamInputTypeInput  StreamInputType = "input"
	StreamInputTypeResize StreamInputType = "resize"
	StreamInputTypeSignal StreamInputType = "signal"
)

// StreamInput is a WebSocket control message sent to a context.
type StreamInput struct {
	Type      StreamInputType `json:"type,omitempty"`
	Data      string          `json:"data,omitempty"`
	Rows      uint16          `json:"rows,omitempty"`
	Cols      uint16          `json:"cols,omitempty"`
	Signal    string          `json:"signal,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
}

// StreamOutput is a WebSocket output message from a context.
type StreamOutput struct {
	SandboxID string
	ContextID string
	Source    string `json:"source"`
	Data      string `json:"data"`
}

type streamOutputMessage struct {
	Source string `json:"source"`
	Data   string `json:"data"`
}

// RunStream opens a REPL context WebSocket stream.
func (s *Sandbox) RunStream(ctx context.Context, language string, input <-chan StreamInput, opts ...RunOption) (<-chan StreamOutput, <-chan error, func() error, error) {
	options := runOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	contextID, err := s.ensureReplContext(ctx, language, options)
	if err != nil {
		return nil, nil, nil, err
	}

	conn, _, err := s.Contexts.ConnectWS(ctx, contextID)
	if err != nil {
		return nil, nil, nil, err
	}

	outputs, errs, closeFn := s.streamContext(ctx, contextID, conn, input, nil)
	return outputs, errs, closeFn, nil
}

// CmdStream opens a CMD context WebSocket stream.
func (s *Sandbox) CmdStream(ctx context.Context, cmd string, input <-chan StreamInput, opts ...CmdOption) (<-chan StreamOutput, <-chan error, func() error, error) {
	if strings.TrimSpace(cmd) == "" {
		return nil, nil, nil, errors.New("command cannot be empty")
	}

	options := cmdOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if options.command == nil {
		parsed, err := parseCommand(cmd)
		if err != nil {
			return nil, nil, nil, err
		}
		options.command = parsed
	}
	if len(options.command) == 0 {
		return nil, nil, nil, errors.New("command cannot be empty")
	}

	waitUntilDone := false
	contextResp, err := s.Contexts.Create(ctx, apispec.CreateContextRequest{
		Type: ptrProcessType(apispec.Cmd),
		Cmd: &apispec.CreateCMDContextRequest{
			Command: &options.command,
		},
		Cwd:            options.cwd,
		EnvVars:        options.envVars,
		PtySize:        options.ptySize,
		IdleTimeoutSec: options.idleTimeoutSec,
		TtlSec:         options.ttlSec,
		WaitUntilDone:  &waitUntilDone,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	if contextResp == nil {
		return nil, nil, nil, errors.New("create context returned nil response")
	}

	conn, _, err := s.Contexts.ConnectWS(ctx, contextResp.Id)
	if err != nil {
		cleanupContext(s, contextResp.Id)
		return nil, nil, nil, err
	}

	outputs, errs, closeFn := s.streamContext(ctx, contextResp.Id, conn, input, func() {
		cleanupContext(s, contextResp.Id)
	})
	return outputs, errs, closeFn, nil
}

func cleanupContext(s *Sandbox, contextID string) {
	if contextID == "" {
		return
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = s.Contexts.Delete(cleanupCtx, contextID)
}

func normalizeStreamInput(input StreamInput) (StreamInput, error) {
	if input.Type == "" {
		input.Type = StreamInputTypeInput
	}
	switch input.Type {
	case StreamInputTypeInput:
		if input.RequestID == "" {
			input.RequestID = generateRequestID()
		}
	case StreamInputTypeResize:
		if input.Rows == 0 || input.Cols == 0 {
			return input, errors.New("resize rows and cols must be > 0")
		}
	case StreamInputTypeSignal:
		if strings.TrimSpace(input.Signal) == "" {
			return input, errors.New("signal is required")
		}
	default:
		return input, fmt.Errorf("unsupported stream input type: %s", input.Type)
	}
	return input, nil
}

func (s *Sandbox) streamContext(
	ctx context.Context,
	contextID string,
	conn *websocket.Conn,
	input <-chan StreamInput,
	onClose func(),
) (<-chan StreamOutput, <-chan error, func() error) {
	outputs := make(chan StreamOutput, 32)
	errs := make(chan error, 1)
	done := make(chan struct{})

	var closeOnce sync.Once
	var closeErr error
	closeStream := func() error {
		closeOnce.Do(func() {
			close(done)
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(time.Second),
			)
			closeErr = conn.Close()
			if onClose != nil {
				onClose()
			}
		})
		return closeErr
	}

	sendErr := func(err error) {
		if err == nil {
			return
		}
		select {
		case errs <- err:
		default:
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			var msg streamOutputMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if ctx.Err() == nil && !isStreamClosed(err, done) {
					sendErr(err)
				}
				closeStream()
				return
			}
			output := StreamOutput{
				SandboxID: s.ID,
				ContextID: contextID,
				Source:    msg.Source,
				Data:      msg.Data,
			}
			select {
			case outputs <- output:
			case <-done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	if input != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				case <-ctx.Done():
					closeStream()
					return
				case msg, ok := <-input:
					if !ok {
						return
					}
					normalized, err := normalizeStreamInput(msg)
					if err != nil {
						sendErr(err)
						continue
					}
					if err := conn.WriteJSON(normalized); err != nil {
						if ctx.Err() == nil {
							sendErr(err)
						}
						closeStream()
						return
					}
				}
			}
		}()
	}

	go func() {
		<-ctx.Done()
		closeStream()
	}()

	go func() {
		wg.Wait()
		close(outputs)
		close(errs)
	}()

	return outputs, errs, closeStream
}

func isStreamClosed(err error, done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	return websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway)
}
