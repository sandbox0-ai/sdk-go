//go:build e2e

package sandbox0_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ogen-go/ogen/ogenerrors"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

type e2eConfig struct {
	baseURL  string
	email    string
	password string
	template string
}

func loadE2EConfig(t *testing.T) e2eConfig {
	t.Helper()
	baseURL := strings.TrimSpace(os.Getenv("S0_E2E_BASE_URL"))
	if baseURL == "" {
		t.Skip("S0_E2E_BASE_URL not set")
	}
	password := strings.TrimSpace(os.Getenv("S0_E2E_PASSWORD"))
	if password == "" {
		t.Skip("S0_E2E_PASSWORD not set")
	}
	email := strings.TrimSpace(os.Getenv("S0_E2E_EMAIL"))
	if email == "" {
		email = "admin@example.com"
	}
	template := strings.TrimSpace(os.Getenv("S0_E2E_TEMPLATE"))
	if template == "" {
		template = "default"
	}
	return e2eConfig{
		baseURL:  baseURL,
		email:    email,
		password: password,
		template: template,
	}
}

type e2eNoAuthSource struct{}

func (e2eNoAuthSource) BearerAuth(ctx context.Context, _ apispec.OperationName) (apispec.BearerAuth, error) {
	return apispec.BearerAuth{}, ogenerrors.ErrSkipClientSecurity
}

func loginWithRetry(ctx context.Context, baseURL, email, password string) (string, error) {
	apiClient, err := apispec.NewClient(baseURL, e2eNoAuthSource{})
	if err != nil {
		return "", err
	}

	deadline := time.Now().Add(15 * time.Second)
	for {
		token, err := loginOnce(ctx, apiClient, email, password)
		if err == nil && strings.TrimSpace(token) != "" {
			return token, nil
		}
		if time.Now().After(deadline) {
			if err == nil {
				err = errors.New("login timeout")
			}
			return "", err
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func loginOnce(ctx context.Context, apiClient *apispec.Client, email, password string) (string, error) {
	resp, err := apiClient.AuthLoginPost(ctx, &apispec.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return "", err
	}
	switch response := resp.(type) {
	case *apispec.SuccessLoginResponse:
		data, ok := response.Data.Get()
		if !ok || strings.TrimSpace(data.AccessToken) == "" {
			return "", errors.New("login response missing token")
		}
		return data.AccessToken, nil
	default:
		return "", errors.New("unexpected login response")
	}
}

func e2eToken(t *testing.T, cfg e2eConfig) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	token, err := loginWithRetry(ctx, cfg.baseURL, cfg.email, cfg.password)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	return token
}

func newClientWithToken(t *testing.T, cfg e2eConfig, token string, opts ...sandbox0.Option) *sandbox0.Client {
	t.Helper()
	allOpts := append([]sandbox0.Option{
		sandbox0.WithBaseURL(cfg.baseURL),
		sandbox0.WithToken(token),
		sandbox0.WithTimeout(30 * time.Second),
	}, opts...)
	client, err := sandbox0.NewClient(allOpts...)
	if err != nil {
		t.Fatalf("create client failed: %v", err)
	}
	return client
}

func claimSandbox(t *testing.T, client *sandbox0.Client, cfg e2eConfig, opts ...sandbox0.SandboxOption) *sandbox0.Sandbox {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	sandbox, err := client.ClaimSandbox(ctx, cfg.template, opts...)
	if err != nil {
		t.Fatalf("claim sandbox failed: %v", err)
	}
	if sandbox == nil || sandbox.ID == "" {
		t.Fatalf("claim sandbox returned empty sandbox")
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteSandbox(cleanupCtx, sandbox.ID)
	})
	return sandbox
}
