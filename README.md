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

## Examples

Runnable examples are available in the `examples/` directory:

| Example                    | Description                              |
|----------------------------|------------------------------------------|
| `01_hello_world`           | Basic sandbox usage                      |
| `02_context_options`       | Context configuration options            |
| `03_files`                 | File read/write/list operations          |
| `04_streaming`             | Streaming execution output               |
| `05_templates`             | Working with sandbox templates           |
| `06_volumes`               | Persistent volumes and snapshots         |
| `07_webhook`               | Webhook event delivery                   |
| `08_network`               | Network policy configuration             |
| `09_expose_port`           | Exposing ports publicly                  |

Run an example:

```bash
cd examples/01_hello_world
SANDBOX0_TOKEN=your-token go run main.go
```

## Links

- [Documentation](https://sandbox0.ai/docs)
- [API Reference](https://sandbox0.ai/docs/api)
- [GitHub Repository](https://github.com/sandbox0-ai/sdk-go)

## License

Apache-2.0
