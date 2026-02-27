package main

import (
	"context"
	"fmt"
	"os"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create a client with auth (and optional base URL).
	client, err := sandbox0.NewClient(
		sandbox0.WithToken(os.Getenv("SANDBOX0_TOKEN")),
		sandbox0.WithBaseURL(os.Getenv("SANDBOX0_BASE_URL")),
	)
	must(err)

	// Claim a sandbox from a template and ensure cleanup.
	sandbox, err := client.ClaimSandbox(ctx, "default", sandbox0.WithSandboxHardTTL(300))
	must(err)
	defer func() {
		if _, err := client.DeleteSandbox(ctx, sandbox.ID); err != nil {
			fmt.Printf("cleanup delete sandbox %s: %v\n", sandbox.ID, err)
		}
	}()

	// Create a custom REPL context with specific cwd, env vars, and TTL.
	// This is useful when you need fine-grained control over the context settings.
	customCtx, err := sandbox.CreateContext(ctx, apispec.CreateContextRequest{
		Type: apispec.NewOptProcessType(apispec.ProcessTypeRepl),
		Repl: apispec.NewOptCreateREPLContextRequest(apispec.CreateREPLContextRequest{
			Alias: apispec.NewOptString("python"),
		}),
		Cwd:            apispec.NewOptString("/workspace"),
		EnvVars:        apispec.NewOptCreateContextRequestEnvVars(map[string]string{"GREETING": "hello from repl"}),
		TTLSec:         apispec.NewOptInt32(120),
		IdleTimeoutSec: apispec.NewOptInt32(60),
	})
	must(err)
	fmt.Printf("Created context: %s\n", customCtx.ID)

	// Run using the custom context via WithContextID.
	runResult, err := sandbox.Run(
		ctx,
		"python",
		`import os, pathlib;
print(pathlib.Path.cwd());
print(os.getenv("GREETING"))`,
		sandbox0.WithContextID(customCtx.ID),
	)
	must(err)
	fmt.Print(runResult.OutputRaw)

	// Run a one-shot command with its own context options.
	// Cmd always creates a new context, so options work directly.
	cmdResult, err := sandbox.Cmd(
		ctx,
		"bash -c 'echo $GREETING && pwd'",
		sandbox0.WithCmdCWD("/tmp"),
		sandbox0.WithCmdEnvVars(map[string]string{"GREETING": "hello from cmd"}),
		sandbox0.WithCmdTTL(120),
		sandbox0.WithCmdIdleTimeout(60),
	)
	must(err)
	fmt.Printf("cmd output:\n%s", cmdResult.OutputRaw)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
