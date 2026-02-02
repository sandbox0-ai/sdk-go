package sandbox0

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type routeMap map[string]http.HandlerFunc

func routeKey(method, path string) string {
	return method + " " + path
}

func newTestServer(t *testing.T, routes routeMap) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler, ok := routes[routeKey(r.Method, r.URL.Path)]; ok {
			handler(w, r)
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
}

func newTestClient(t *testing.T, server *httptest.Server, opts ...Option) *Client {
	t.Helper()
	allOpts := append([]Option{WithBaseURL(server.URL)}, opts...)
	client, err := NewClient(allOpts...)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write JSON response: %v", err)
	}
}
