package main

import (
	"context"
	"fmt"
	"os"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
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
	sandbox, err := client.ClaimSandbox(ctx, "default",
		sandbox0.WithSandboxAutoResume(true),
	)
	must(err)
	// defer client.DeleteSandbox(ctx, sandbox.ID)

	// Upload a simple HTML file to serve
	htmlContent := `<html><body><h1>Hello from Sandbox0!</h1></body></html>`
	_, err = sandbox.WriteFile(ctx, "/tmp/index.html", []byte(htmlContent))
	must(err)
	fmt.Println("Uploaded index.html via file upload API")

	// Start a persistent web server process using Cmd with async mode
	// The process will continue running even after the call returns
	fmt.Println("Starting web server on port 8080...")
	serverResult, err := sandbox.Cmd(ctx,
		"python3 -m http.server 8080 --directory /tmp",
		sandbox0.WithCmdWait(false), // async - don't wait for completion
	)
	must(err)
	fmt.Printf("Server started, context: %s\n", serverResult.ContextID)
	// defer sandbox.DeleteContext(ctx, serverResult.ContextID)

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Expose port 8080 to make it publicly accessible
	fmt.Println("Exposing port 8080...")
	portsResp, err := sandbox.ExposePort(ctx, 8080, true)
	must(err)
	fmt.Printf("Exposure domain: %s\n", portsResp.ExposureDomain)
	fmt.Printf("Exposed ports:\n")
	for _, p := range portsResp.Ports {
		fmt.Printf("  - Port: %d, Resume: %v, PublicURL: %s\n", p.Port, p.Resume, p.PublicURL)
	}

	// Verify the exposed port by making a request from inside the sandbox
	fmt.Println("Verifying server is running...")
	resp, err := sandbox.Cmd(ctx, "curl -s http://localhost:8080/index.html")
	must(err)
	fmt.Printf("Server response: \n%s\n", resp.OutputRaw)

	// List all exposed ports
	allPorts, err := sandbox.GetExposedPorts(ctx)
	must(err)
	fmt.Printf("All exposed ports:\n")
	for _, p := range allPorts.Ports {
		fmt.Printf("  - Port: %d, Resume: %v, PublicURL: %s\n", p.Port, p.Resume, p.PublicURL)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
