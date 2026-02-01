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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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

	// Create a volume and ensure cleanup.
	volume, err := client.Volumes.Create(ctx, apispec.CreateSandboxVolumeRequest{})
	must(err)
	volumeID := volume.Id
	defer client.Volumes.Delete(ctx, volumeID)

	// Mount the volume into the sandbox, write a file, then unmount.
	_, err = client.Volumes.Mount(ctx, sandbox.ID, volumeID, "/mnt/data", nil)
	must(err)
	defer client.Volumes.Unmount(ctx, sandbox.ID, volumeID)

	_, err = sandbox.Files.Write(ctx, "/mnt/data/hello.txt", []byte("hello volume\n"))
	must(err)

	// Create and list snapshots for the volume.
	snapshotName := fmt.Sprintf("snap-%d", time.Now().Unix())
	snapshot, err := client.Volumes.CreateSnapshot(ctx, volumeID, apispec.CreateSnapshotRequest{
		Name: snapshotName,
	})
	must(err)
	fmt.Printf("snapshot created: %s\n", snapshot.Id)

	snapshots, err := client.Volumes.ListSnapshots(ctx, volumeID)
	must(err)
	fmt.Printf("snapshots: %d\n", len(snapshots))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
