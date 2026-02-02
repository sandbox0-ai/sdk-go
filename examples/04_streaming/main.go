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
	sandbox, err := client.ClaimSandbox(ctx, "default")
	must(err)
	defer client.DeleteSandbox(ctx, sandbox.ID)

	fmt.Println("REPL stream:")
	replInput := make(chan sandbox0.StreamInput)
	replOutputs, replErrs, closeRepl, err := sandbox.RunStream(ctx, "python", replInput)
	must(err)

	go func() {
		replInput <- sandbox0.StreamInput{Type: sandbox0.StreamInputTypeInput, Data: "print('hello from repl')"}
		replInput <- sandbox0.StreamInput{Type: sandbox0.StreamInputTypeInput, Data: "print(1 + 2)"}
		close(replInput)
		time.AfterFunc(500*time.Millisecond, func() {
			_ = closeRepl()
		})
	}()

	must(readStream(ctx, replOutputs, replErrs))

	fmt.Println("CMD stream:")
	cmdOutputs, cmdErrs, closeCmd, err := sandbox.CmdStream(ctx, `bash -c "for i in 1 2 3; do echo line-$i; done"`, nil)
	must(err)
	time.AfterFunc(2*time.Second, func() {
		_ = closeCmd()
	})

	must(readStream(ctx, cmdOutputs, cmdErrs))
}

func readStream(ctx context.Context, outputs <-chan sandbox0.StreamOutput, errs <-chan error) error {
	for {
		select {
		case output, ok := <-outputs:
			if !ok {
				return nil
			}
			fmt.Print(output.Data)
		case err, ok := <-errs:
			if ok && err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
