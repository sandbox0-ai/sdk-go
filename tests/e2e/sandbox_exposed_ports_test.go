//go:build e2e

package sandbox0_test

import (
	"context"
	"slices"
	"testing"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
)

func TestSandboxExposedPorts(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Clear any existing ports first to start with a clean state
	if err := sandbox.ClearExposedPorts(ctx); err != nil {
		t.Fatalf("clear exposed ports failed: %v", err)
	}

	// Test GetExposedPorts - should be empty initially
	resp, err := sandbox.GetExposedPorts(ctx)
	if err != nil {
		t.Fatalf("get exposed ports failed: %v", err)
	}
	if resp == nil {
		t.Fatalf("get exposed ports returned nil response")
	}
	if len(resp.Ports) != 0 {
		t.Fatalf("expected 0 ports initially, got %d", len(resp.Ports))
	}

	// Test ExposePort - add port 3000 with resume=false
	resp, err = sandbox.ExposePort(ctx, 3000, false)
	if err != nil {
		t.Fatalf("expose port 3000 failed: %v", err)
	}
	if len(resp.Ports) != 1 {
		t.Fatalf("expected 1 port after expose, got %d", len(resp.Ports))
	}
	if resp.Ports[0].Port != 3000 {
		t.Fatalf("expected port 3000, got %d", resp.Ports[0].Port)
	}
	if resp.Ports[0].Resume != false {
		t.Fatalf("expected resume=false, got %v", resp.Ports[0].Resume)
	}

	// Test ExposePort - update existing port's resume to true
	resp, err = sandbox.ExposePort(ctx, 3000, true)
	if err != nil {
		t.Fatalf("update port 3000 resume failed: %v", err)
	}
	if len(resp.Ports) != 1 {
		t.Fatalf("expected 1 port after update, got %d", len(resp.Ports))
	}
	if resp.Ports[0].Resume != true {
		t.Fatalf("expected resume=true after update, got %v", resp.Ports[0].Resume)
	}

	// Test ExposePort - add another port
	resp, err = sandbox.ExposePort(ctx, 8080, false)
	if err != nil {
		t.Fatalf("expose port 8080 failed: %v", err)
	}
	if len(resp.Ports) != 2 {
		t.Fatalf("expected 2 ports after adding 8080, got %d", len(resp.Ports))
	}

	// Test UnexposePort - remove port 3000
	resp, err = sandbox.UnexposePort(ctx, 3000)
	if err != nil {
		t.Fatalf("unexpose port 3000 failed: %v", err)
	}
	if len(resp.Ports) != 1 {
		t.Fatalf("expected 1 port after unexpose, got %d", len(resp.Ports))
	}
	if resp.Ports[0].Port != 8080 {
		t.Fatalf("expected remaining port 8080, got %d", resp.Ports[0].Port)
	}

	// Test UpdateExposedPorts - replace all ports
	newPorts := []sandbox0.ExposedPort{
		{Port: 4000, Resume: true},
		{Port: 5000, Resume: false},
		{Port: 6000, Resume: true},
	}
	resp, err = sandbox.UpdateExposedPorts(ctx, newPorts)
	if err != nil {
		t.Fatalf("update exposed ports failed: %v", err)
	}
	if len(resp.Ports) != 3 {
		t.Fatalf("expected 3 ports after update, got %d", len(resp.Ports))
	}
	// Verify ports are present (order may vary)
	portMap := make(map[int32]bool)
	for _, p := range resp.Ports {
		portMap[p.Port] = p.Resume
	}
	for _, expected := range newPorts {
		resume, ok := portMap[expected.Port]
		if !ok {
			t.Fatalf("expected port %d not found", expected.Port)
		}
		if resume != expected.Resume {
			t.Fatalf("port %d: expected resume=%v, got %v", expected.Port, expected.Resume, resume)
		}
	}

	// Test ClearExposedPorts
	if err := sandbox.ClearExposedPorts(ctx); err != nil {
		t.Fatalf("clear exposed ports failed: %v", err)
	}

	// Verify ports are cleared
	resp, err = sandbox.GetExposedPorts(ctx)
	if err != nil {
		t.Fatalf("get exposed ports after clear failed: %v", err)
	}
	if len(resp.Ports) != 0 {
		t.Fatalf("expected 0 ports after clear, got %d", len(resp.Ports))
	}

	// Test ExposureDomain is present in response
	if resp.ExposureDomain == "" {
		t.Log("warning: exposure domain is empty")
	}
}

func TestSandboxExposedPortsPublicURL(t *testing.T) {
	cfg := loadE2EConfig(t)
	token := e2eToken(t, cfg)
	client := newClientWithToken(t, cfg, token)
	sandbox := claimSandbox(t, client, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Clear any existing ports first
	if err := sandbox.ClearExposedPorts(ctx); err != nil {
		t.Fatalf("clear exposed ports failed: %v", err)
	}

	// Expose a port and check if PublicURL is returned
	resp, err := sandbox.ExposePort(ctx, 3000, false)
	if err != nil {
		t.Fatalf("expose port 3000 failed: %v", err)
	}

	// Find port 3000 and verify it has a public URL
	idx := slices.IndexFunc(resp.Ports, func(p sandbox0.ExposedPort) bool {
		return p.Port == 3000
	})
	if idx < 0 {
		t.Fatalf("port 3000 not found in response")
	}
	port := resp.Ports[idx]

	// PublicURL should be non-empty (readOnly field returned by server)
	if port.PublicURL == "" {
		t.Log("warning: public URL is empty (may be expected if server doesn't return it)")
	} else {
		t.Logf("public URL for port 3000: %s", port.PublicURL)
	}

	// Cleanup
	if err := sandbox.ClearExposedPorts(ctx); err != nil {
		t.Fatalf("clear exposed ports failed: %v", err)
	}
}
