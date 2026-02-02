package sandbox0

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/shlex"
	"github.com/gorilla/websocket"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// Sandbox is a convenience wrapper for sandbox-scoped operations.
type Sandbox struct {
	ID        string
	Template  string
	ClusterID *string
	PodName   string
	Status    string

	client            *Client
	replContextByLang map[string]string
	mu                sync.Mutex
}

// RunResult represents REPL execution output.
type RunResult struct {
	SandboxID string
	ContextID string
	Output    string
}

// CmdResult represents CMD execution output.
type CmdResult struct {
	SandboxID string
	ContextID string
	Output    string
}

type runOptions struct {
	contextID      string
	idleTimeoutSec *int32
	ttlSec         *int32
	cwd            *string
	envVars        *map[string]string
	ptySize        *apispec.PTYSize
}

// RunOption configures sandbox Run behavior.
type RunOption func(*runOptions)

// WithContextID uses a specific context ID.
func WithContextID(contextID string) RunOption {
	return func(opts *runOptions) {
		opts.contextID = contextID
	}
}

// WithContextTTL sets TTL in seconds for created contexts.
func WithContextTTL(ttlSec int32) RunOption {
	return func(opts *runOptions) {
		opts.ttlSec = &ttlSec
	}
}

// WithIdleTimeout sets idle timeout in seconds for created contexts.
func WithIdleTimeout(idleTimeoutSec int32) RunOption {
	return func(opts *runOptions) {
		opts.idleTimeoutSec = &idleTimeoutSec
	}
}

// WithCWD sets the working directory for created contexts.
func WithCWD(cwd string) RunOption {
	return func(opts *runOptions) {
		opts.cwd = &cwd
	}
}

// WithEnvVars sets environment variables for created contexts.
func WithEnvVars(envVars map[string]string) RunOption {
	return func(opts *runOptions) {
		opts.envVars = &envVars
	}
}

// WithPTYSize sets PTY size for created contexts.
func WithPTYSize(rows, cols uint16) RunOption {
	return func(opts *runOptions) {
		rows32 := int32(rows)
		cols32 := int32(cols)
		opts.ptySize = &apispec.PTYSize{Rows: &rows32, Cols: &cols32}
	}
}

// Run executes input in a REPL context.
func (s *Sandbox) Run(ctx context.Context, language, input string, opts ...RunOption) (RunResult, error) {
	if strings.TrimSpace(input) == "" {
		return RunResult{}, errors.New("input cannot be empty")
	}

	options := runOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	contextID, err := s.ensureReplContext(ctx, language, options)
	if err != nil {
		return RunResult{}, err
	}

	execResp, err := s.ContextExec(ctx, contextID, input)
	if err != nil {
		return RunResult{}, err
	}

	return RunResult{
		SandboxID: s.ID,
		ContextID: contextID,
		Output:    execResp.Output,
	}, nil
}

type cmdOptions struct {
	command        []string
	idleTimeoutSec *int32
	ttlSec         *int32
	cwd            *string
	envVars        *map[string]string
	ptySize        *apispec.PTYSize
}

// CmdOption configures sandbox Cmd behavior.
type CmdOption func(*cmdOptions)

// WithCommand overrides the command used by Cmd.
func WithCommand(command []string) CmdOption {
	return func(opts *cmdOptions) {
		opts.command = command
	}
}

// WithCmdTTL sets TTL in seconds for created CMD contexts.
func WithCmdTTL(ttlSec int32) CmdOption {
	return func(opts *cmdOptions) {
		opts.ttlSec = &ttlSec
	}
}

// WithCmdIdleTimeout sets idle timeout in seconds for created CMD contexts.
func WithCmdIdleTimeout(idleTimeoutSec int32) CmdOption {
	return func(opts *cmdOptions) {
		opts.idleTimeoutSec = &idleTimeoutSec
	}
}

// WithCmdCWD sets the working directory for created CMD contexts.
func WithCmdCWD(cwd string) CmdOption {
	return func(opts *cmdOptions) {
		opts.cwd = &cwd
	}
}

// WithCmdEnvVars sets environment variables for created CMD contexts.
func WithCmdEnvVars(envVars map[string]string) CmdOption {
	return func(opts *cmdOptions) {
		opts.envVars = &envVars
	}
}

// WithCmdPTYSize sets PTY size for created CMD contexts.
func WithCmdPTYSize(rows, cols uint16) CmdOption {
	return func(opts *cmdOptions) {
		rows32 := int32(rows)
		cols32 := int32(cols)
		opts.ptySize = &apispec.PTYSize{Rows: &rows32, Cols: &cols32}
	}
}

// Cmd executes a one-time command in a CMD context.
func (s *Sandbox) Cmd(ctx context.Context, cmd string, opts ...CmdOption) (CmdResult, error) {
	if strings.TrimSpace(cmd) == "" {
		return CmdResult{}, errors.New("command cannot be empty")
	}

	options := cmdOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if options.command == nil {
		parsed, err := parseCommand(cmd)
		if err != nil {
			return CmdResult{}, err
		}
		options.command = parsed
	}
	if len(options.command) == 0 {
		return CmdResult{}, errors.New("command cannot be empty")
	}

	waitUntilDone := true
	contextResp, err := s.CreateContext(ctx, apispec.CreateContextRequest{
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
		return CmdResult{}, err
	}
	if contextResp == nil {
		return CmdResult{}, errors.New("create context returned nil response")
	}
	defer s.DeleteContext(ctx, contextResp.Id)

	output := ""
	if contextResp.Output != nil {
		output = *contextResp.Output
	}

	return CmdResult{
		SandboxID: s.ID,
		ContextID: contextResp.Id,
		Output:    output,
	}, nil
}

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

	conn, _, err := s.ConnectWSContext(ctx, contextID)
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
	contextResp, err := s.CreateContext(ctx, apispec.CreateContextRequest{
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

	conn, _, err := s.ConnectWSContext(ctx, contextResp.Id)
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
	_, _ = s.DeleteContext(cleanupCtx, contextID)
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

func (s *Sandbox) ensureReplContext(ctx context.Context, language string, options runOptions) (string, error) {
	if options.contextID != "" {
		return options.contextID, nil
	}

	language = strings.TrimSpace(language)
	if language == "" {
		language = "python"
	}

	s.mu.Lock()
	contextID := s.replContextByLang[language]
	s.mu.Unlock()
	if contextID != "" {
		return contextID, nil
	}

	contextResp, err := s.CreateContext(ctx, apispec.CreateContextRequest{
		Type: ptrProcessType(apispec.Repl),
		Repl: &apispec.CreateREPLContextRequest{
			Language: &language,
		},
		Cwd:            options.cwd,
		EnvVars:        options.envVars,
		PtySize:        options.ptySize,
		IdleTimeoutSec: options.idleTimeoutSec,
		TtlSec:         options.ttlSec,
	})
	if err != nil {
		return "", err
	}
	if contextResp == nil {
		return "", errors.New("create context returned nil response")
	}

	contextID = contextResp.Id
	s.mu.Lock()
	s.replContextByLang[language] = contextID
	s.mu.Unlock()

	return contextID, nil
}

func ptrProcessType(value apispec.ProcessType) *apispec.ProcessType {
	return &value
}

func parseCommand(input string) ([]string, error) {
	args, err := shlex.Split(input)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, errors.New("command cannot be empty")
	}
	return args, nil
}

var requestCounter atomic.Uint64

func generateRequestID() string {
	count := requestCounter.Add(1)
	return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), count)
}
