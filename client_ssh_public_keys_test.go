package sandbox0

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestClientSSHPublicKeys(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/users/me/ssh-keys" {
				t.Fatalf("path = %s, want /users/me/ssh-keys", r.URL.Path)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"ssh_keys": []map[string]any{
						{
							"id":                 "key_123",
							"name":               "macbook",
							"public_key":         "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey user@example",
							"key_type":           "ssh-ed25519",
							"fingerprint_sha256": "SHA256:example",
							"comment":            "user@example",
							"created_at":         "2026-04-10T12:00:00Z",
							"updated_at":         "2026-04-10T12:00:00Z",
						},
					},
				},
			})
		})
		defer server.Close()

		keys, err := client.ListUserSSHPublicKeys(context.Background())
		if err != nil {
			t.Fatalf("ListUserSSHPublicKeys() error = %v", err)
		}
		if len(keys) != 1 {
			t.Fatalf("len(keys) = %d, want 1", len(keys))
		}
		if keys[0].Name != "macbook" {
			t.Fatalf("key name = %q, want macbook", keys[0].Name)
		}
		if keys[0].FingerprintSHA256 != "SHA256:example" {
			t.Fatalf("fingerprint = %q, want SHA256:example", keys[0].FingerprintSHA256)
		}
	})

	t.Run("create", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/users/me/ssh-keys" {
				t.Fatalf("path = %s, want /users/me/ssh-keys", r.URL.Path)
			}
			body := decodeCreateSSHPublicKeyRequest(t, r.Body)
			if body.Name != "macbook" {
				t.Fatalf("request name = %q, want macbook", body.Name)
			}
			if body.PublicKey != "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey user@example" {
				t.Fatalf("public key = %q", body.PublicKey)
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"id":                 "key_123",
					"name":               body.Name,
					"public_key":         body.PublicKey,
					"key_type":           "ssh-ed25519",
					"fingerprint_sha256": "SHA256:example",
					"comment":            "user@example",
					"created_at":         "2026-04-10T12:00:00Z",
					"updated_at":         "2026-04-10T12:00:00Z",
				},
			})
		})
		defer server.Close()

		key, err := client.CreateUserSSHPublicKey(context.Background(), apispec.CreateSSHPublicKeyRequest{
			Name:      "macbook",
			PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey user@example",
		})
		if err != nil {
			t.Fatalf("CreateUserSSHPublicKey() error = %v", err)
		}
		if key.Name != "macbook" {
			t.Fatalf("key name = %q, want macbook", key.Name)
		}
	})

	t.Run("delete", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s, want DELETE", r.Method)
			}
			if r.URL.Path != "/users/me/ssh-keys/key_123" {
				t.Fatalf("path = %s, want /users/me/ssh-keys/key_123", r.URL.Path)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"message": "deleted",
				},
			})
		})
		defer server.Close()

		resp, err := client.DeleteUserSSHPublicKey(context.Background(), "key_123")
		if err != nil {
			t.Fatalf("DeleteUserSSHPublicKey() error = %v", err)
		}
		data, ok := resp.Data.Get()
		if !ok {
			t.Fatal("response data not set")
		}
		message, ok := data.Message.Get()
		if !ok || message != "deleted" {
			t.Fatalf("message = %q, want deleted", message)
		}
	})

	t.Run("list not found surfaces api error", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusNotFound, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "not_found",
					"message": "user not found",
				},
			})
		})
		defer server.Close()

		_, err := client.ListUserSSHPublicKeys(context.Background())
		if err == nil {
			t.Fatal("ListUserSSHPublicKeys() error = nil, want error")
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("error type = %T, want *APIError", err)
		}
		if apiErr.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", apiErr.StatusCode)
		}
	})
}

func decodeCreateSSHPublicKeyRequest(t *testing.T, body io.Reader) apispec.CreateSSHPublicKeyRequest {
	t.Helper()

	var req apispec.CreateSSHPublicKeyRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return req
}
