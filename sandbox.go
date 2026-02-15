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
	OutputRaw string
}

// CmdResult represents CMD execution output.
type CmdResult struct {
	SandboxID string
	ContextID string
	OutputRaw string
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
		opts.ptySize = &apispec.PTYSize{
			Rows: apispec.NewOptInt32(rows32),
			Cols: apispec.NewOptInt32(cols32),
		}
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
	if options.wait != nil {
		waitUntilDone = *options.wait
	}
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

	return CmdResult{
		SandboxID: s.ID,
		ContextID: contextResp.ID,
		OutputRaw: outputRaw,
	}, nil
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

	req := apispec.CreateContextRequest{
		Type: apispec.NewOptProcessType(apispec.ProcessTypeRepl),
		Repl: apispec.NewOptCreateREPLContextRequest(apispec.CreateREPLContextRequest{
			Language: apispec.NewOptString(language),
		}),
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
	contextResp, err := s.CreateContext(ctx, req)
	if err != nil {
		return "", err
	}
	if contextResp == nil {
		return "", errors.New("create context returned nil response")
	}

	contextID = contextResp.ID
	s.mu.Lock()
	s.replContextByLang[language] = contextID
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
