package sandbox0

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestGetRegistryCredentials(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/registry/credentials" {
			t.Fatalf("path = %s, want /api/v1/registry/credentials", r.URL.Path)
		}
		var request apispec.RegistryCredentialsRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if got, ok := request.TargetImage.Get(); !ok || got != "image-builds/build-123:latest" {
			t.Fatalf("targetImage = %q, %v", got, ok)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"provider":     "builtin",
				"pushRegistry": "registry.local/team",
				"pullRegistry": "registry.pull/team",
				"username":     "builder",
				"password":     "secret",
			},
		})
	})
	defer server.Close()

	credentials, err := client.GetRegistryCredentials(context.Background(), "image-builds/build-123:latest")
	if err != nil {
		t.Fatalf("GetRegistryCredentials() error = %v", err)
	}
	if credentials.PushRegistry != "registry.local/team" || credentials.Username != "builder" {
		t.Fatalf("credentials = %#v", credentials)
	}
}
