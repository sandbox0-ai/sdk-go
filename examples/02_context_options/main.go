package main

import (
	"context"
	"fmt"
	"os"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
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
	sandbox, err := client.Sandboxes.Claim(ctx, "default")
	must(err)
	defer client.Sandboxes.Delete(ctx, sandbox.ID)

	// Run with REPL options: working dir, env, temporary context, TTL/idle timeout.
	runResult, err := sandbox.Run(
		ctx,
		"python",
		`import os, pathlib;
print(pathlib.Path.cwd());
print(os.getenv("GREETING"))`,
		sandbox0.WithCWD("/tmp"),
		sandbox0.WithEnvVars(map[string]string{"GREETING": "hello from repl"}),
		sandbox0.WithContextTTL(120),
		sandbox0.WithIdleTimeout(60),
	)
	must(err)
	fmt.Print(runResult.Output)

	// Run a one-shot command with its own context options.
	cmdResult, err := sandbox.Cmd(
		ctx,
		"bash -c 'echo $GREETING && pwd'",
		sandbox0.WithCmdCWD("/tmp"),
		sandbox0.WithCmdEnvVars(map[string]string{"GREETING": "hello from cmd"}),
		sandbox0.WithCmdTTL(120),
		sandbox0.WithCmdIdleTimeout(60),
	)
	must(err)
	fmt.Printf("cmd output:\n%s", cmdResult.Output)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
