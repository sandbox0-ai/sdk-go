package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	webhookURL := os.Getenv("SANDBOX0_WEBHOOK_URL")
	if webhookURL == "" {
		// Tip: use https://webhook.site to get a temporary URL for debugging.
		log.Fatal("SANDBOX0_WEBHOOK_URL is required")
	}
	webhookSecret := os.Getenv("SANDBOX0_WEBHOOK_SECRET")

	client, err := sandbox0.NewClient(
		sandbox0.WithToken(os.Getenv("SANDBOX0_TOKEN")),
		sandbox0.WithBaseURL(os.Getenv("SANDBOX0_BASE_URL")),
	)
	must(err)

	baseDir := "/tmp33/webhook-demo"
	sandbox, err := client.ClaimSandbox(
		ctx,
		"default",
		sandbox0.WithSandboxHardTTL(300),
		sandbox0.WithSandboxWebhook(webhookURL, webhookSecret),
		sandbox0.WithSandboxWebhookWatchDir(baseDir),
	)
	must(err)
	defer func() {
		if _, err := client.DeleteSandbox(ctx, sandbox.ID); err != nil {
			log.Printf("cleanup delete sandbox %s: %v", sandbox.ID, err)
		}
	}()

	// Process started/exited events (exit code 0).
	_, err = sandbox.Run(ctx, "bash", `echo webhook test`)
	must(err)
	log.Println("process started")

	// Sandbox paused/resumed events.
	_, err = client.PauseSandbox(ctx, sandbox.ID)
	must(err)
	log.Println("sandbox paused")
	_, err = client.ResumeSandbox(ctx, sandbox.ID)
	must(err)
	log.Println("sandbox resumed")

	// Process crashed event (non-zero exit code).
	_, err = sandbox.Cmd(ctx, `/bin/sh -c "exit 2"`)
	if err != nil {
		// Non-zero exit is expected; still proceed.
		log.Printf("expected command error: %v", err)
	}
	log.Println("process crashed")

	// File modified events: create/write/rename/chmod/remove.
	_, err = sandbox.Mkdir(ctx, baseDir, true)
	must(err)
	_, err = sandbox.WriteFile(ctx, baseDir+"/file.txt", []byte("hello"))
	must(err)
	_, err = sandbox.MoveFile(ctx, baseDir+"/file.txt", baseDir+"/file-renamed.txt")
	must(err)
	_, err = sandbox.Cmd(ctx,
		fmt.Sprintf(`/bin/sh -c "chmod 600 %s/file-renamed.txt"`, strings.TrimRight(baseDir, "/")))
	must(err)
	_, err = sandbox.DeleteFile(ctx, baseDir+"/file-renamed.txt")
	must(err)
	log.Println("file modified")
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
