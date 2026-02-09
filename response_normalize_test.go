package sandbox0

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestNormalizeNullMapResponse(t *testing.T) {
	payload := map[string]any{
		"annotations": nil,
		"entries":     nil,
		"env_vars": map[string]any{
			"FOO": nil,
		},
	}
	raw, _ := json.Marshal(payload)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(raw)),
	}
	if err := normalizeNullMapResponse(context.Background(), resp); err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	var normalized map[string]any
	if err := json.Unmarshal(body, &normalized); err != nil {
		t.Fatalf("unmarshal normalized: %v", err)
	}
	if _, ok := normalized["annotations"].(map[string]any); !ok {
		t.Fatalf("expected annotations to be map, got %T", normalized["annotations"])
	}
	if _, ok := normalized["entries"].([]any); !ok {
		t.Fatalf("expected entries to be array, got %T", normalized["entries"])
	}
	envVars, ok := normalized["env_vars"].(map[string]any)
	if !ok || envVars["FOO"] != "" {
		t.Fatalf("expected env_vars.FOO to be empty string, got %v", envVars["FOO"])
	}
}

func TestNormalizeNullMapsNoChange(t *testing.T) {
	payload := any(map[string]any{"foo": "bar"})
	if normalizeNullMaps(&payload) {
		t.Fatal("expected no changes")
	}
}
