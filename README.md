# Sandbox0 Go SDK

The official Go SDK for Sandbox0, providing typed models and ergonomic high-level APIs for managing secure code execution sandboxes.

> This SDK remains open source. Current Sandbox0 runtime development is closed source.
> Sandbox0 Cloud is the supported service. Self-hosting is no longer supported.
> Bring Your Own Cloud (BYOC) is planned and is not available yet.
> See [deployment options](https://sandbox0.ai/docs/sandbox/deployment).

## Installation

```bash
go get github.com/sandbox0-ai/sdk-go
```

## Requirements

- Go 1.25 or later

## RootFS capacity

Template `resources.ephemeralStorage` sets the initial RootFS filesystem capacity
for image imports: the default is `8Gi`, and supported values are `300Mi` through
`1Ti`, aligned to 4096 bytes. On deployments with automatic growth enabled, XFS
expands online within the configured sparse-device ceiling while the sandbox's
processes continue running. Existing generations retain their committed geometry.

Persistent blocks reside in regional object storage. Compute nodes retain bounded
caches and write journals; slow or unavailable storage can apply write backpressure.
The initial size is distinct from stored data usage and service storage quotas.
Automatic growth requires operator enablement; see the
[deployment guide](https://github.com/sandbox0-ai/sandbox0/blob/main/deploy/nomad/rootfs-autogrow.md).

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

## Independent sandbox health (unreleased)

```go
health, err := client.GetSandboxHealth(ctx, sandboxID)
if err != nil { return err }
fmt.Println(health.Runtime.Liveness.State, health.Execution.State, health.Terminal.State)
```

The caller's context bounds the request. Generations remain exact `int64` values.

Health is returned as separate runtime launch/liveness, execution, terminal,
and storage identity/writer/I/O observations. Each includes `state`, a fixed
`reason`, its evidence `source`, and the original observation time (null when
unobserved). Guest evidence is advisory. A storage identity or writer lease does
not prove current storage I/O; that check currently remains `unknown`.

The query may run bounded fixed guest diagnostics, but never resumes or restarts
a sandbox. Paused sandboxes and active lifecycle transitions skip those probes.
Unavailable guest diagnostics are a successful observation response, not an API
failure or a reason to retry a lifecycle mutation. This API requires a server
version with the health endpoint; SDKs do not fall back to guest commands.

## Links

- [Documentation](https://sandbox0.ai/docs)
- [API Reference](https://sandbox0.ai/docs/api)
- [GitHub Repository](https://github.com/sandbox0-ai/sdk-go)

## License

Apache-2.0


Conditional session receipts can be read without auto-resume or process creation:
`sandbox.FindConditionalSession(ctx, originalKey, generation)`.
This uses the distinct `/sessions/conditional/lookup` route and checks the original
generation. Errors never fall back to ordinary lookup or creation. A missing
receipt does not authorize replay: an earlier create may still be in flight.
The result is process information, not credential/storage qualification or proof
that the process is still running when a later action begins.

### Conditional Session attachment

`Sandbox.ConnectConditionalSession(ctx, sessionID, key, generation, attemptID, options)` opens the distinct `/sessions/{session_id}/conditional/ws` endpoint, bound to the original conditional creation key, runtime generation and attempt. It accepts the existing Session WebSocket replay options. Every control message is checked against that binding, and generation handover revokes the connection. Errors never trigger an ordinary attachment, retry or auto-resume. Closing the connection detaches without stopping the process. This is not lifecycle, storage or credential qualification.

### Reconnect an explicitly memory-restored original attempt

```go
session, err := sandbox.FindRestoredConditionalSession(ctx, originalKey, currentGeneration, creationGeneration)
connection, response, err := sandbox.ConnectRestoredConditionalSession(ctx, session.ID, originalKey, currentGeneration, creationGeneration, originalAttemptID, options)
```

Verify the external memory-restoration lineage and your owned credential/session
history first. These helpers explicitly send `creation_runtime_generation`, which
must precede the current generation. They require the original still-running
attempt and never create a session, restart an attempt, resume the sandbox or
retry through ordinary endpoints. Existing same-generation helpers remain strict.
The restored socket can replay the original attempt's earlier-generation events;
its controls are bound to the current generation and the original attempt ID.
A subsequent handover revokes the socket. Old gateways that discard the optional
origin cannot authorize a restored original creation and must return an error.

## Explicit snapshot recovery

`RestoreSandboxSnapshot(ctx, sandboxID, apispec.SandboxSnapshotRestoreRequest{ExpectedRuntimeGeneration: 7, SnapshotID: snapshotID, SnapshotGenerationID: snapshotGenerationID, DiscardProcessState: true}, requestKey)` explicitly replaces writable RootFS and ends old processes. Keep every request field and the key after an uncertain reply. Use `FindSandboxSnapshotRestoreOperation` or `GetSandboxSnapshotRestoreOperation` to observe the original operation without resubmitting. Only `succeeded` proves fresh runtime launch; `RestoredRootfsGenerationID` alone proves the head change committed.

`GetSandboxSnapshotRestoreFinalCapture`, `ReadSandboxSnapshotRestoreFinalCaptureFile` and `AcknowledgeSandboxSnapshotRestoreFinalCapture` provide the same stopped-source credential custody protocol on restore-specific routes. Acknowledge only after durable encrypted owner commit. These helpers never retry, refresh coordinates or fall back to a guest command or disk-only memory recovery.

## Read original RootFS fork provenance

```go
receipt, err := client.GetSandboxRootFSFork(ctx, sourceSandboxID, 7, "original-fork")
```

Keep the original source sandbox, generation (including zero), key and
authenticated identity. This performs one GET and never retries or submits a
fork. Missing, conflicting and pending results remain errors; a 404 is not
proof of cancellation. The receipt requires the original unchanged, never-run
paused child and retained storage evidence. Memory forks are excluded. It is
not a credential-safety attestation or a state reservation; later mutations
need their own identity conditions.

## Conditional first launch of a RootFS fork

```go
original := sandbox0.RootFSForkLaunchOptions{
    ForkOperationID: receipt.OperationID, FilesystemID: receipt.TargetFilesystemID,
    HeadGenerationID: receipt.TargetHeadGenerationID, WriterEpoch: receipt.WriterEpoch,
    IdempotencyKey: "original-child-launch",
} // Persist these conditions before submitting.
launch, err := client.LaunchSandboxRootFSFork(ctx, receipt.TargetSandboxID, original)
if err != nil { return err } // Retain original even when the reply is lost.
original.ExpectedOperationID = launch.OperationID
observation, err := client.GetSandboxRootFSForkLaunch(ctx, receipt.TargetSandboxID, original)
```

Each method sends exactly one request. After a lost reply, query with the same
original conditions and key, leaving `ExpectedOperationID` empty if no receipt
was received. Never refresh the original writer epoch or fall back to ordinary
resume. Completed history retains its digest and time even when `Current` is
false after later lifecycle changes. `Current` is an owner snapshot, not a live
lease, health check or credential authorization. Pending has no completion
proof; neither aborted nor 404 proves physical absence. Helpers validate the
returned identity and state-specific evidence and never poll or retry for you.
