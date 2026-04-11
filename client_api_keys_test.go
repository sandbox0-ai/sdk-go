package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestClientGetCurrentAPIKey(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api-keys/current" {
				t.Fatalf("path = %s, want /api-keys/current", r.URL.Path)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
				t.Fatalf("authorization = %q, want %q", got, "Bearer test-token")
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"api_key": map[string]any{
						"id":          "key_123",
						"team_id":     "team_123",
						"created_by":  "user_123",
						"type":        "secret",
						"roles":       []string{"sandbox:create"},
						"permissions": []string{"sandbox:create"},
						"is_active":   true,
						"expires_at":  "2026-04-10T00:00:00Z",
					},
				},
			})
		})
		defer server.Close()

		identity, err := client.GetCurrentAPIKey(context.Background())
		if err != nil {
			t.Fatalf("GetCurrentAPIKey() error = %v", err)
		}
		if identity.ID != "key_123" || identity.TeamID != "team_123" {
			t.Fatalf("identity = %#v", identity)
		}
		if identity.CreatedBy == nil || *identity.CreatedBy != "user_123" {
			t.Fatalf("created_by = %#v, want user_123", identity.CreatedBy)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"code": "unauthorized", "message": "unauthorized"},
			})
		})
		defer server.Close()

		if _, err := client.GetCurrentAPIKey(context.Background()); err == nil {
			t.Fatal("GetCurrentAPIKey() error = nil, want error")
		}
	})
}
