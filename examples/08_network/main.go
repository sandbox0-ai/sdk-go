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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create a client with auth (and optional base URL).
	client, err := sandbox0.NewClient(
		sandbox0.WithToken(os.Getenv("SANDBOX0_TOKEN")),
		sandbox0.WithBaseURL(os.Getenv("SANDBOX0_BASE_URL")),
	)
	must(err)

	// Claim a sandbox from a template and ensure cleanup.
	sandbox, err := client.ClaimSandbox(ctx, "default",
		sandbox0.WithSandboxHardTTL(600),
		sandbox0.WithSandboxNetworkPolicy(apispec.TplSandboxNetworkPolicy{
			Mode: apispec.TplSandboxNetworkPolicyModeAllowAll,
		}),
	)
	must(err)
	defer client.DeleteSandbox(ctx, sandbox.ID)

	current, err := sandbox.GetNetworkPolicy(ctx)
	must(err)
	fmt.Printf("current policy: %+v\n", current)

	const shell = `/bin/curl -s -o /dev/null -w "%{http_code}\n" --max-time 3 https://github.com`
	resp, err := sandbox.Cmd(ctx, shell)
	must(err)
	fmt.Println(resp.Output)

	// Block all traffic
	_, err = sandbox.UpdateNetworkPolicy(ctx, apispec.TplSandboxNetworkPolicy{
		Mode: apispec.TplSandboxNetworkPolicyModeBlockAll,
	})
	must(err)

	resp, err = sandbox.Cmd(ctx, shell)
	must(err)
	fmt.Println(resp.Output)

	_, err = sandbox.UpdateNetworkPolicy(ctx, apispec.TplSandboxNetworkPolicy{
		Mode: apispec.TplSandboxNetworkPolicyModeBlockAll,
		Egress: apispec.NewOptNetworkEgressPolicy(apispec.NetworkEgressPolicy{
			AllowedDomains: []string{"github.com"},
		}),
	})
	must(err)

	resp, err = sandbox.Cmd(ctx, shell)
	must(err)
	fmt.Println(resp.Output)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
