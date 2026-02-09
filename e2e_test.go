//go:build e2e

package sandbox0

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

type e2eNoAuthSource struct{}

func (e2eNoAuthSource) BearerAuth(ctx context.Context, _ apispec.OperationName) (apispec.BearerAuth, error) {
	return apispec.BearerAuth{}, ogenerrors.ErrSkipClientSecurity
}

func TestE2EClaimSandbox(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("S0_E2E_BASE_URL"))
	if baseURL == "" {
		t.Skip("S0_E2E_BASE_URL not set")
	}
	password := os.Getenv("S0_E2E_PASSWORD")
	if strings.TrimSpace(password) == "" {
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

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	token, err := loginWithRetry(ctx, baseURL, email, password)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	client, err := NewClient(
		WithBaseURL(baseURL),
		WithToken(token),
		WithTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("create client failed: %v", err)
	}

	templates, err := client.ListTemplate(ctx)
	if err != nil {
		t.Fatalf("list templates failed: %v", err)
	}
	if len(templates) == 0 {
		t.Fatalf("no templates available")
	}

	sandbox, err := client.ClaimSandbox(ctx, template, WithSandboxHardTTL(300))
	if err != nil {
		t.Fatalf("claim sandbox failed: %v", err)
	}
	if sandbox == nil || sandbox.ID == "" {
		t.Fatalf("claim sandbox returned empty sandbox")
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = client.DeleteSandbox(cleanupCtx, sandbox.ID)
	}()

	if _, err := client.GetSandbox(ctx, sandbox.ID); err != nil {
		t.Fatalf("get sandbox failed: %v", err)
	}
	if _, err := client.StatusSandbox(ctx, sandbox.ID); err != nil {
		t.Fatalf("status sandbox failed: %v", err)
	}
}

func loginWithRetry(ctx context.Context, baseURL, email, password string) (string, error) {
	apiClient, err := apispec.NewClient(baseURL, e2eNoAuthSource{})
	if err != nil {
		return "", err
	}

	deadline := time.Now().Add(3 * time.Minute)
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
		case <-time.After(5 * time.Second):
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
		return "", apiErrorFromResponse(response)
	}
}
