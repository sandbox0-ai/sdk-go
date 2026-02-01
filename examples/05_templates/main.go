package main

import (
	"context"
	"fmt"
	"os"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a client with auth (and optional base URL).
	client, err := sandbox0.NewClient(
		sandbox0.WithToken(os.Getenv("SANDBOX0_TOKEN")),
		sandbox0.WithBaseURL(os.Getenv("SANDBOX0_BASE_URL")),
	)
	must(err)

	// List templates available for sandbox creation.
	templates, err := client.Templates.List(ctx)
	must(err)
	fmt.Printf("templates: %d\n", len(templates))
	for _, tpl := range templates {
		name := ""
		display := ""
		if tpl.Metadata != nil && tpl.Metadata.Name != nil {
			name = *tpl.Metadata.Name
		}
		if tpl.Spec != nil && tpl.Spec.DisplayName != nil {
			display = *tpl.Spec.DisplayName
		}
		fmt.Printf("- name=%s display=%s\n", name, display)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
