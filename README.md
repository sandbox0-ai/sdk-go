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

## Documentation

- [Sandbox0 docs](https://sandbox0.ai/docs)
- [Volume mounts](https://sandbox0.ai/docs/volume/mounts)

## Bootstrap Mounts At Claim Time

You can request existing volumes during `ClaimSandbox` so the sandbox starts with them already mounted:

```go
volume, err := client.CreateVolume(ctx, apispec.CreateSandboxVolumeRequest{})
if err != nil {
    log.Fatal(err)
}

sandbox, err := client.ClaimSandbox(
    ctx,
    "default",
    sandbox0.WithSandboxBootstrapMount(volume.ID, "/workspace/data"),
)
if err != nil {
    log.Fatal(err)
}

for _, mount := range sandbox.BootstrapMounts {
    fmt.Printf("%s %s\n", mount.SandboxvolumeID, mount.State)
}
```

## Links

- [Documentation](https://sandbox0.ai/docs)
- [API Reference](https://sandbox0.ai/docs/api)
- [GitHub Repository](https://github.com/sandbox0-ai/sdk-go)

## License

Apache-2.0
