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
	sandbox, err := client.ClaimSandbox(ctx, "default", sandbox0.WithSandboxHardTTL(300))
	must(err)
	defer func() {
		if _, err := client.DeleteSandbox(ctx, sandbox.ID); err != nil {
			fmt.Printf("cleanup delete sandbox %s: %v\n", sandbox.ID, err)
		}
	}()

	// Create a volume and ensure cleanup.
	volume, err := client.CreateVolume(ctx, apispec.CreateSandboxVolumeRequest{
		AccessMode: apispec.NewOptVolumeAccessMode(apispec.VolumeAccessModeRWX),
	})
	must(err)
	volumeID := volume.ID
	defer func() {
		if _, err := client.DeleteVolume(ctx, volumeID); err != nil {
			fmt.Printf("cleanup delete volume %s: %v\n", volumeID, err)
		}
	}()
	fmt.Printf("volume created: %s\n", volumeID)

	// Mount the volume into the sandbox, write a file, then unmount.
	mountResp, err := sandbox.Mount(ctx, volumeID, "/mnt/data", nil)
	must(err)
	defer func() {
		if _, err := sandbox.Unmount(ctx, volumeID, mountResp.MountSessionID); err != nil {
			fmt.Printf("cleanup unmount volume %s in sandbox %s: %v\n", volumeID, sandbox.ID, err)
		}
	}()
	fmt.Printf("volume mounted: %s\n", volumeID)

	_, err = sandbox.WriteFile(ctx, "/mnt/data/hello.txt", []byte("hello volume\n"))
	must(err)
	fmt.Printf("file written: /mnt/data/hello.txt\n")

	// Create snapshot for the volume.
	snapshotName := fmt.Sprintf("snap-%d", time.Now().Unix())
	snapshot, err := client.CreateVolumeSnapshot(ctx, volumeID, apispec.CreateSnapshotRequest{
		Name: snapshotName,
	})
	must(err)
	fmt.Printf("snapshot created: %s\n", snapshot.ID)

	// Update the file in the volume.
	_, err = sandbox.WriteFile(ctx, "/mnt/data/hello.txt", []byte("hello volume\nsecond line\n"))
	must(err)
	fmt.Printf("file updated: /mnt/data/hello.txt\n")

	readResult, err := sandbox.ReadFile(ctx, "/mnt/data/hello.txt")
	must(err)
	fmt.Printf("file content: \n%s", string(readResult))

	// Restore the snapshot.
	_, err = client.RestoreVolumeSnapshot(ctx, volumeID, snapshot.ID)
	must(err)
	fmt.Printf("snapshot restored: %s\n", snapshot.ID)

	readResult, err = sandbox.ReadFile(ctx, "/mnt/data/hello.txt")
	must(err)
	fmt.Printf("file content: \n%s", string(readResult))

	// Fork the volume and verify COW isolation.
	forkedVolume, err := client.ForkVolume(ctx, volumeID, &apispec.ForkVolumeRequest{})
	must(err)
	forkedVolumeID := forkedVolume.ID
	defer func() {
		if _, err := client.DeleteVolume(ctx, forkedVolumeID); err != nil {
			fmt.Printf("cleanup delete forked volume %s: %v\n", forkedVolumeID, err)
		}
	}()
	fmt.Printf("volume forked: %s (source: %s)\n", forkedVolumeID, volumeID)

	// Create a new sandbox
	sandbox2, err := client.ClaimSandbox(ctx, "default")
	must(err)
	defer func() {
		if _, err := client.DeleteSandbox(ctx, sandbox2.ID); err != nil {
			fmt.Printf("cleanup delete sandbox %s: %v\n", sandbox2.ID, err)
		}
	}()
	fmt.Printf("new sandbox created: %s\n", sandbox2.ID)

	mountResp2, err := sandbox2.Mount(ctx, volumeID, "/mnt/data", nil)
	must(err)
	defer func() {
		if _, err := sandbox2.Unmount(ctx, volumeID, mountResp2.MountSessionID); err != nil {
			fmt.Printf("cleanup unmount volume %s in sandbox %s: %v\n", volumeID, sandbox2.ID, err)
		}
	}()

	readResult, err = sandbox2.ReadFile(ctx, "/mnt/data/hello.txt")
	must(err)
	fmt.Printf("sandbox2 file content: \n%s", string(readResult))

	mountRespFork, err := sandbox2.Mount(ctx, forkedVolumeID, "/mnt/fork", nil)
	must(err)
	defer func() {
		if _, err := sandbox2.Unmount(ctx, forkedVolumeID, mountRespFork.MountSessionID); err != nil {
			fmt.Printf("cleanup unmount forked volume %s in sandbox %s: %v\n", forkedVolumeID, sandbox2.ID, err)
		}
	}()

	_, err = sandbox2.WriteFile(ctx, "/mnt/fork/hello.txt", []byte("hello from fork\n"))
	must(err)

	forkContent, err := sandbox2.ReadFile(ctx, "/mnt/fork/hello.txt")
	must(err)
	fmt.Printf("forked volume file content: \n%s", string(forkContent))

	sourceContent, err := sandbox2.ReadFile(ctx, "/mnt/data/hello.txt")
	must(err)
	fmt.Printf("source volume file content after fork write: \n%s", string(sourceContent))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
