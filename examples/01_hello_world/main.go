package main

import (
	"context"
	"fmt"
	"log"
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
	sandbox, err := client.ClaimSandbox(ctx, "default", sandbox0.WithSandboxHardTTL(300))
	must(err)
	defer func() {
		if _, err := client.DeleteSandbox(ctx, sandbox.ID); err != nil {
			log.Printf("cleanup delete sandbox %s: %v", sandbox.ID, err)
		}
	}()

	// Run a REPL-style snippet (stateful; env/vars preserved between Run calls).
	runResult, err := sandbox.Run(ctx, "python", `x=2`)
	must(err)
	fmt.Print(runResult.OutputRaw)

	runResult, err = sandbox.Run(ctx, "python", `print(x)`)
	must(err)
	fmt.Print(runResult.OutputRaw)

	// Run a one-shot command (stateless; env/vars not preserved between Cmd calls).
	fmt.Println("\nRunning command: /bin/sh -c \"x=3\"")
	_, err = sandbox.Cmd(ctx, `/bin/sh -c "x=3"`)
	must(err)

	fmt.Println("Running command: /bin/sh -c \"echo $x\"")
	cmdResult, err := sandbox.Cmd(ctx, `/bin/sh -c "echo $x"`)
	must(err)
	fmt.Print(cmdResult.OutputRaw)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
