# Sandbox0 Go SDK

The official Go SDK for Sandbox0, providing typed models and ergonomic high-level APIs for managing secure code execution sandboxes.

## Installation

```bash
go get github.com/sandbox0-ai/sdk-go
```

## Requirements

- Go 1.25 or later

## Configuration

| Environment Variable | Required | Default                   | Description          |
|---------------------|----------|---------------------------|----------------------|
| `SANDBOX0_TOKEN`    | Yes      | -                         | API authentication token |
| `SANDBOX0_BASE_URL` | No       | `https://api.sandbox0.ai` | API base URL         |

## Quick Start

```go
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

    // Create a client
    client, err := sandbox0.NewClient(
        sandbox0.WithToken(os.Getenv("SANDBOX0_TOKEN")),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Claim a sandbox
    sandbox, err := client.ClaimSandbox(ctx, "default")
    if err != nil {
        log.Fatal(err)
    }
    defer client.DeleteSandbox(ctx, sandbox.ID)

    // Execute Python code (REPL - stateful)
    result, err := sandbox.Run(ctx, "python", "print('Hello, Sandbox0!')")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(result.OutputRaw)
}
```

## CMD Streaming

```go
stream, err := sandbox.CmdStream(
    ctx,
    "sh -c 'echo hello && echo warn >&2'",
    sandbox0.WithCommand([]string{"sh", "-c", "echo hello && echo warn >&2"}),
)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    output, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(output.Data)
}

done, err := stream.Wait()
if err != nil {
    log.Fatal(err)
}
fmt.Printf("exit=%d state=%s\n", *done.ExitCode, done.State)
```

## Usage Windows

Usage windows are immutable, team-scoped usage records. Retain `NextCursor` to
incrementally import only newly recorded windows:

```go
page, err := client.ListUsageWindows(ctx, &sandbox0.ListUsageWindowsOptions{
    Cursor:     savedCursor,
    Limit:      250,
    WindowType: "sandbox.runtime_mib_milliseconds",
})
if err != nil {
    log.Fatal(err)
}

for _, window := range page.Windows {
    fmt.Printf("%s %d %s\n", window.WindowID, window.Value, window.Unit)
}
savedCursor = page.NextCursor
```

## Documentation

- [Sandbox0 docs](https://sandbox0.ai/docs)

## Create A Template From A Sandbox

Capture an initialized sandbox root filesystem as a team-owned template. Creation is asynchronous; the capture point is `status.creation.capturedAt`, not request acceptance, so keep the source sandbox available until capture completes. A running source is briefly write-barriered and remains running afterward. `WaitTemplateReady` polls until the immutable RootFS base is ready for claims. Canceling the context only stops polling and does not cancel the server-side build.

```go
request := sandbox0.NewTemplateFromSandboxCreateRequest(
    "python-workspace",
    sandbox.ID,
    &apispec.TemplateFromSandboxSpecOverrides{
        DisplayName: apispec.NewOptString("Python workspace"),
        Tags:        []string{"python"},
    },
)

template, err := client.CreateTemplateFromSandbox(
    ctx,
    request,
    &sandbox0.CreateTemplateFromSandboxOptions{
        IdempotencyKey: "python-workspace-v1",
    },
)
if err != nil {
    log.Fatal(err)
}

template, err = client.WaitTemplateReady(ctx, template.TemplateID, nil)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("template %s is ready\n", template.TemplateID)
```

## Links

- [Documentation](https://sandbox0.ai/docs)
- [API Reference](https://sandbox0.ai/docs/api)
- [GitHub Repository](https://github.com/sandbox0-ai/sdk-go)

## License

Apache-2.0
