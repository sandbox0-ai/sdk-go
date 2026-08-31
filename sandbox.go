package sandbox0

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/google/shlex"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// Sandbox is a convenience wrapper for sandbox-scoped operations.
type Sandbox struct {
	ID        string
	Template  string
	ClusterID *string
	RuntimeID string
	Status    string

	client            *Client
	replContextByLang map[string]string
	mu                sync.Mutex
}

// RunResult represents REPL execution output.
type RunResult struct {
	SandboxID string
	ContextID string
	OutputRaw string
}

// CmdResult represents CMD execution output.
type CmdResult struct {
	SandboxID string
	ContextID string
	OutputRaw string
	Stdout    string
	Stderr    string
	ExitCode  *int
	State     string
}

type runOptions struct {
	contextID string
}

// RunOption configures sandbox Run behavior.
type RunOption func(*runOptions)

// WithContextID uses a specific context ID instead of the default cached REPL context.
// Use this when you need custom envVars, cwd, or other context settings.
func WithContextID(contextID string) RunOption {
	return func(opts *runOptions) {
		opts.contextID = contextID
	}
}

// Run executes input in a REPL context.
func (s *Sandbox) Run(ctx context.Context, alias, input string, opts ...RunOption) (RunResult, error) {
	if strings.TrimSpace(input) == "" {
		return RunResult{}, errors.New("input cannot be empty")
	}

	options := runOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	contextID, err := s.ensureReplContext(ctx, alias, options)
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
		OutputRaw: execResp.OutputRaw,
	}, nil
}

type cmdOptions struct {
	command        []string
	wait           *bool
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

// WithCmdWait sets whether to wait for command completion.
// Default is true. Set to false for async execution.
func WithCmdWait(wait bool) CmdOption {
	return func(opts *cmdOptions) {
		opts.wait = &wait
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
		opts.ptySize = &apispec.PTYSize{
			Rows: apispec.NewOptInt32(rows32),
			Cols: apispec.NewOptInt32(cols32),
		}
	}
}

// Cmd executes a command in a CMD context.
// By default, it waits for command completion. Use WithCmdWait(false) for async execution.
// The context is not automatically deleted; use DeleteContext to clean up when done.
func (s *Sandbox) Cmd(ctx context.Context, cmd string, opts ...CmdOption) (CmdResult, error) {
	options, err := resolveCmdOptions(cmd, opts...)
	if err != nil {
		return CmdResult{}, err
	}

	waitUntilDone := true
	if options.wait != nil {
		waitUntilDone = *options.wait
	}
	req := buildCmdCreateContextRequest(options, waitUntilDone)
	contextResp, err := s.CreateContext(ctx, req)
	if err != nil {
		return CmdResult{}, err
	}
	if contextResp == nil {
		return CmdResult{}, errors.New("create context returned nil response")
	}

	outputRaw := ""
	if value, ok := contextResp.OutputRaw.Get(); ok {
		outputRaw = value
	}
	stdout := ""
	if value, ok := contextResp.Stdout.Get(); ok {
		stdout = value
	}
	stderr := ""
	if value, ok := contextResp.Stderr.Get(); ok {
		stderr = value
	}

	return CmdResult{
		SandboxID: s.ID,
		ContextID: contextResp.ID,
		OutputRaw: outputRaw,
		Stdout:    stdout,
		Stderr:    stderr,
		ExitCode:  optInt32ToIntPtr(contextResp.ExitCode),
		State:     optStringValue(contextResp.State),
	}, nil
}

// CmdStream executes a command in a CMD context and returns a connected WebSocket stream.
func (s *Sandbox) CmdStream(ctx context.Context, cmd string, opts ...CmdOption) (*ContextStream, error) {
	options, err := resolveCmdOptions(cmd, opts...)
	if err != nil {
		return nil, err
	}
	if options.wait != nil && *options.wait {
		return nil, errors.New("cmd stream requires wait=false")
	}

	contextResp, err := s.CreateContext(ctx, buildCmdCreateContextRequest(options, false))
	if err != nil {
		return nil, err
	}
	if contextResp == nil {
		return nil, errors.New("create context returned nil response")
	}

	conn, _, err := s.ConnectWSContext(ctx, contextResp.ID)
	if err != nil {
		return nil, err
	}
	return newContextStream(s.ID, contextResp.ID, conn), nil
}

func resolveCmdOptions(cmd string, opts ...CmdOption) (cmdOptions, error) {
	if strings.TrimSpace(cmd) == "" {
		return cmdOptions{}, errors.New("command cannot be empty")
	}

	options := cmdOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	if options.command == nil {
		parsed, err := parseCommand(cmd)
		if err != nil {
			return cmdOptions{}, err
		}
		options.command = parsed
	}
	if len(options.command) == 0 {
		return cmdOptions{}, errors.New("command cannot be empty")
	}

	return options, nil
}

func buildCmdCreateContextRequest(options cmdOptions, waitUntilDone bool) apispec.CreateContextRequest {
	req := apispec.CreateContextRequest{
		Type:          apispec.NewOptProcessType(apispec.ProcessTypeCmd),
		Cmd:           apispec.NewOptCreateCMDContextRequest(apispec.CreateCMDContextRequest{Command: options.command}),
		WaitUntilDone: apispec.NewOptBool(waitUntilDone),
	}
	if options.cwd != nil {
		req.Cwd = apispec.NewOptString(*options.cwd)
	}
	if options.envVars != nil {
		req.EnvVars = apispec.NewOptCreateContextRequestEnvVars(apispec.CreateContextRequestEnvVars(*options.envVars))
	}
	if options.ptySize != nil {
		req.PtySize = apispec.NewOptPTYSize(*options.ptySize)
	}
	if options.idleTimeoutSec != nil {
		req.IdleTimeoutSec = apispec.NewOptInt32(*options.idleTimeoutSec)
	}
	if options.ttlSec != nil {
		req.TTLSec = apispec.NewOptInt32(*options.ttlSec)
	}
	return req
}

func optInt32ToIntPtr(value apispec.OptInt32) *int {
	raw, ok := value.Get()
	if !ok {
		return nil
	}
	converted := int(raw)
	return &converted
}

func optStringValue(value apispec.OptString) string {
	raw, ok := value.Get()
	if !ok {
		return ""
	}
	return raw
}

func (s *Sandbox) ensureReplContext(ctx context.Context, alias string, options runOptions) (string, error) {
	if options.contextID != "" {
		return options.contextID, nil
	}

	alias = strings.TrimSpace(alias)
	if alias == "" {
		alias = "python"
	}

	s.mu.Lock()
	contextID := s.replContextByLang[alias]
	s.mu.Unlock()
	if contextID != "" {
		return contextID, nil
	}

	// Create a default REPL context with no custom settings
	req := apispec.CreateContextRequest{
		Type: apispec.NewOptProcessType(apispec.ProcessTypeRepl),
		Repl: apispec.NewOptCreateREPLContextRequest(apispec.CreateREPLContextRequest{
			Alias: apispec.NewOptString(alias),
		}),
	}
	contextResp, err := s.CreateContext(ctx, req)
	if err != nil {
		return "", err
	}
	if contextResp == nil {
		return "", errors.New("create context returned nil response")
	}

	contextID = contextResp.ID
	s.mu.Lock()
	s.replContextByLang[alias] = contextID
	s.mu.Unlock()

	return contextID, nil
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
