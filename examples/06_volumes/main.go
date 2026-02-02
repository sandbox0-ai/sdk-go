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
	fmt.Printf("volume created: %s\n", volumeID)

	// Mount the volume into the sandbox, write a file, then unmount.
	_, err = client.Volumes.Mount(ctx, sandbox.ID, volumeID, "/mnt/data", nil)
	must(err)
	defer client.Volumes.Unmount(ctx, sandbox.ID, volumeID)
	fmt.Printf("volume mounted: %s\n", volumeID)

	_, err = sandbox.Files.Write(ctx, "/mnt/data/hello.txt", []byte("hello volume\n"))
	must(err)
	fmt.Printf("file written: /mnt/data/hello.txt\n")

	// Create snapshot for the volume.
	snapshotName := fmt.Sprintf("snap-%d", time.Now().Unix())
	snapshot, err := client.Volumes.CreateSnapshot(ctx, volumeID, apispec.CreateSnapshotRequest{
		Name: snapshotName,
	})
	must(err)
	fmt.Printf("snapshot created: %s\n", snapshot.Id)

	// Update the file in the volume.
	_, err = sandbox.Files.Write(ctx, "/mnt/data/hello.txt", []byte("hello volume\nsecond line\n"))
	must(err)
	fmt.Printf("file updated: /mnt/data/hello.txt\n")

	readResult, err := sandbox.Files.Read(ctx, "/mnt/data/hello.txt")
	must(err)
	fmt.Printf("file content: \n%s", string(readResult))

	// Restore the snapshot.
	_, err = client.Volumes.RestoreSnapshot(ctx, volumeID, snapshot.Id)
	must(err)
	fmt.Printf("snapshot restored: %s\n", snapshot.Id)

	readResult, err = sandbox.Files.Read(ctx, "/mnt/data/hello.txt")
	must(err)
	fmt.Printf("file content: \n%s", string(readResult))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
