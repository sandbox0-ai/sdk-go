package main

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	defer client.DeleteSandbox(ctx, sandbox.ID)

	dir := "/tmp/sdk-go"
	path := dir + "/hello.txt"

	// Create a directory, write a file, and read it back.
	_, err = sandbox.Mkdir(ctx, dir, true)
	must(err)
	fmt.Println("file created")

	_, err = sandbox.WriteFile(ctx, path, []byte("hello from file\n"))
	must(err)
	fmt.Println("file written")

	readResult, err := sandbox.ReadFile(ctx, path)
	must(err)
	fmt.Printf("file content: %s\n", strings.TrimSpace(string(readResult)))
	fmt.Println("file read")

	entries, err := sandbox.ListFiles(ctx, dir)
	must(err)
	fmt.Printf("dir entries: %d\n", len(entries))
	for _, entry := range entries {
		if path, ok := entry.Path.Get(); ok {
			fmt.Printf("- %s\n", path)
		}
	}

	// Subscribe to file watch events, then write to trigger one.
	watchCtx, watchCancel := context.WithTimeout(ctx, 10*time.Second)
	defer watchCancel()
	events, errs, unsubscribe, err := sandbox.WatchFiles(watchCtx, dir, true)
	must(err)
	defer unsubscribe()

	_, err = sandbox.WriteFile(ctx, path, []byte("hello from file\nsecond line\n"))
	must(err)

	select {
	case ev, ok := <-events:
		if ok {
			fmt.Printf("watch event: type=%s path=%s event=%s\n", ev.Type, ev.Path, ev.Event)
		}
	case err, ok := <-errs:
		if ok && err != nil {
			must(err)
		}
	case <-watchCtx.Done():
		fmt.Println("watch timeout")
	}

	// Use CMD to read the file from inside the sandbox.
	runResult, err := sandbox.Cmd(ctx, `cat /tmp/sdk-go/hello.txt`)
	must(err)
	fmt.Printf("run output:\n%s", runResult.OutputRaw)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
