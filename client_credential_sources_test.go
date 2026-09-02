package sandbox0

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestClientCredentialSources(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/api/v1/credential-sources" {
				t.Fatalf("path = %s, want /api/v1/credential-sources", r.URL.Path)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": []map[string]any{
					{
						"name":           "source-a",
						"resolverKind":   "static_headers",
						"currentVersion": 3,
						"status":         "active",
					},
				},
			})
		})
		defer server.Close()

		sources, err := client.ListCredentialSources(context.Background())
		if err != nil {
			t.Fatalf("ListCredentialSources() error = %v", err)
		}
		if len(sources) != 1 {
			t.Fatalf("len(sources) = %d, want 1", len(sources))
		}
		if got := sources[0].Name; got != "source-a" {
			t.Fatalf("source name = %q, want source-a", got)
		}
		if got := sources[0].ResolverKind; got != apispec.CredentialSourceResolverKindStaticHeaders {
			t.Fatalf("resolver kind = %q, want %q", got, apispec.CredentialSourceResolverKindStaticHeaders)
		}
	})

	t.Run("get", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/api/v1/credential-sources/source-a" {
				t.Fatalf("path = %s, want /api/v1/credential-sources/source-a", r.URL.Path)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"name":           "source-a",
					"resolverKind":   "static_username_password",
					"currentVersion": 7,
					"status":         "active",
				},
			})
		})
		defer server.Close()

		source, err := client.GetCredentialSource(context.Background(), "source-a")
		if err != nil {
			t.Fatalf("GetCredentialSource() error = %v", err)
		}
		if got := source.Name; got != "source-a" {
			t.Fatalf("source name = %q, want source-a", got)
		}
	})

	t.Run("get not found", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusNotFound, map[string]any{
				"success": false,
				"error": map[string]any{
					"code":    "not_found",
					"message": "credential source not found",
				},
			})
		})
		defer server.Close()

		_, err := client.GetCredentialSource(context.Background(), "missing")
		if err == nil {
			t.Fatal("GetCredentialSource() error = nil, want error")
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("error type = %T, want *APIError", err)
		}
		if apiErr.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", apiErr.StatusCode)
		}
		if apiErr.Code != "not_found" {
			t.Fatalf("code = %q, want not_found", apiErr.Code)
		}
	})

	t.Run("create", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/api/v1/credential-sources" {
				t.Fatalf("path = %s, want /api/v1/credential-sources", r.URL.Path)
			}
			body := decodeCredentialSourceWriteRequest(t, r.Body)
			if body.Name != "source-a" {
				t.Fatalf("request name = %q, want source-a", body.Name)
			}
			staticHeaders, ok := body.Spec.StaticHeaders.Get()
			if !ok {
				t.Fatal("staticHeaders not set")
			}
			values, ok := staticHeaders.Values.Get()
			if !ok {
				t.Fatal("staticHeaders.values not set")
			}
			if values["Authorization"] != "Bearer token" {
				t.Fatalf("header value = %q, want %q", values["Authorization"], "Bearer token")
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"success": true,
				"data": map[string]any{
					"name":           "source-a",
					"resolverKind":   "static_headers",
					"currentVersion": 1,
					"status":         "active",
				},
			})
		})
		defer server.Close()

		source, err := client.CreateCredentialSource(context.Background(), apispec.CredentialSourceWriteRequest{
			Name:         "source-a",
			ResolverKind: apispec.CredentialSourceResolverKindStaticHeaders,
			Spec: apispec.CredentialSourceWriteSpec{
				StaticHeaders: apispec.NewOptStaticHeadersSourceSpec(apispec.StaticHeadersSourceSpec{
					Values: apispec.NewOptStaticHeadersSourceSpecValues(apispec.StaticHeadersSourceSpecValues{
						"Authorization": "Bearer token",
					}),
				}),
			},
		})
		if err != nil {
			t.Fatalf("CreateCredentialSource() error = %v", err)
		}
		if got := source.Name; got != "source-a" {
			t.Fatalf("source name = %q, want source-a", got)
		}
	})

	t.Run("update forces path name", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Fatalf("method = %s, want PUT", r.Method)
			}
			if r.URL.Path != "/api/v1/credential-sources/source-b" {
				t.Fatalf("path = %s, want /api/v1/credential-sources/source-b", r.URL.Path)
			}
			body := decodeCredentialSourceWriteRequest(t, r.Body)
			if body.Name != "source-b" {
				t.Fatalf("request name = %q, want source-b", body.Name)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"name":           "source-b",
					"resolverKind":   "static_username_password",
					"currentVersion": 2,
					"status":         "active",
				},
			})
		})
		defer server.Close()

		source, err := client.UpdateCredentialSource(context.Background(), "source-b", apispec.CredentialSourceWriteRequest{
			Name:         "stale-name",
			ResolverKind: apispec.CredentialSourceResolverKindStaticUsernamePassword,
			Spec: apispec.CredentialSourceWriteSpec{
				StaticUsernamePassword: apispec.NewOptStaticUsernamePasswordSourceSpec(apispec.StaticUsernamePasswordSourceSpec{
					Username: "alice",
					Password: "secret",
				}),
			},
		})
		if err != nil {
			t.Fatalf("UpdateCredentialSource() error = %v", err)
		}
		if got := source.Name; got != "source-b" {
			t.Fatalf("source name = %q, want source-b", got)
		}
	})

	t.Run("delete", func(t *testing.T) {
		client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s, want DELETE", r.Method)
			}
			if r.URL.Path != "/api/v1/credential-sources/source-a" {
				t.Fatalf("path = %s, want /api/v1/credential-sources/source-a", r.URL.Path)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"message": "deleted",
				},
			})
		})
		defer server.Close()

		resp, err := client.DeleteCredentialSource(context.Background(), "source-a")
		if err != nil {
			t.Fatalf("DeleteCredentialSource() error = %v", err)
		}
		data := resp.Data
		message, ok := data.Message.Get()
		if !ok || message != "deleted" {
			t.Fatalf("message = %q, want deleted", message)
		}
	})
}

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)
	client, err := NewClient(
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
		WithToken("test-token"),
	)
	if err != nil {
		server.Close()
		t.Fatalf("NewClient() error = %v", err)
	}
	return client, server
}

func decodeCredentialSourceWriteRequest(t *testing.T, body io.Reader) apispec.CredentialSourceWriteRequest {
	t.Helper()

	var req apispec.CredentialSourceWriteRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return req
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response body: %v", err)
	}
}
