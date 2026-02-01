package sandbox0

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	Contexts          SandboxContextService
	Files             SandboxFileService
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

	execResp, err := s.Contexts.Exec(ctx, contextID, input)
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
		return CmdResult{}, err
	}
	if contextResp == nil {
		return CmdResult{}, errors.New("create context returned nil response")
	}
	defer s.Contexts.Delete(ctx, contextResp.Id)

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

	contextResp, err := s.Contexts.Create(ctx, apispec.CreateContextRequest{
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
