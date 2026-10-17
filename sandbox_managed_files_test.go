package sandbox0

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"net/http"
	"testing"
)

func TestManagedFilesUseGeneratedRoutesAndExactSequence(t *testing.T) {
	for _, namespace := range []string{"", "codex"} {
		t.Run(namespace, func(t *testing.T) {
			content := []byte{0, 255, 128, 10}
			calls := 0
			client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				path := "/api/v1/sandboxes/sb_123/managed-files"
				if calls != 1 {
					if namespace != "" {
						path = "/api/v1/sandboxes/sb_123/managed-file-namespaces/" + namespace
					}
					path += "/auth.json"
				}
				if r.URL.Path != path || r.URL.Query().Get("runtime_generation") != "7" {
					t.Errorf("unexpected route or generation: %s", r.URL)
				}
				state := map[string]any{"runtime_generation": 7, "mutation_sequence": "9007199254740993", "size_bytes": 8 << 20, "guest_path": "/run/sandbox0/credentials"}
				if r.Method == "PUT" {
					var body map[string]string
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if body["expected_mutation_sequence"] != "9007199254740993" || body["content"] != base64.StdEncoding.EncodeToString(content) {
						t.Error("write lost exact bytes or sequence")
					}
				}
				if r.Method == "GET" && calls != 1 {
					state["content"] = base64.StdEncoding.EncodeToString(content)
				}
				if r.Method == "DELETE" && r.URL.Query().Get("expected_mutation_sequence") != "18446744073709551615" {
					t.Error("delete lost uint64 precision")
				}
				writeJSON(t, w, 200, map[string]any{"success": true, "data": state})
			})
			defer server.Close()
			files := client.Sandbox("sb_123").ManagedFiles(7, namespace)
			ctx := context.Background()
			state, err := files.Inspect(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = files.Write(ctx, "auth.json", content, state.MutationSequence); err != nil {
				t.Fatal(err)
			}
			read, err := files.Read(ctx, "auth.json")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(read.Content, content) {
				t.Error("binary content changed")
			}
			if _, err = files.Delete(ctx, "auth.json", "18446744073709551615"); err != nil {
				t.Fatal(err)
			}
			if calls != 4 {
				t.Fatalf("unexpected background requests: %d", calls)
			}
		})
	}
}

func TestManagedFilesDoNotRetryAmbiguousMutationsOrFallBack(t *testing.T) {
	calls := 0
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "PUT" || r.URL.Path != "/api/v1/sandboxes/sb_123/managed-file-namespaces/codex/auth.json" {
			t.Error("unexpected recovery or fallback request")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["content"] != "" {
			t.Error("nil content must encode empty bytes")
		}
		writeJSON(t, w, 503, map[string]any{"success": false, "error": map[string]any{"code": "unavailable", "message": "unavailable"}})
	})
	defer server.Close()
	_, err := client.Sandbox("sb_123").ManagedFiles(7, "codex").Write(context.Background(), "auth.json", nil, "0")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 503 {
		t.Fatalf("expected API error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("mutation retried %d times", calls)
	}
}

func TestManagedFilesRejectInvalidInputsBeforeTransport(t *testing.T) {
	client, server := newTestClient(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid request reached transport") })
	defer server.Close()
	ctx := context.Background()
	for _, name := range []string{"../auth", "a/b", ".s0-write-hidden", ".", "..", "auth?x", "é"} {
		if _, err := client.Sandbox("sb_123").ManagedFiles(7, "codex").Read(ctx, name); err == nil {
			t.Error("accepted invalid name")
		}
		if _, err := client.Sandbox("sb_123").ManagedFiles(7, name).Read(ctx, "auth.json"); err == nil {
			t.Error("accepted invalid namespace")
		}
	}
	for _, seq := range []string{"01", "-1", "18446744073709551616"} {
		if _, err := client.Sandbox("sb_123").ManagedFiles(7, "").Write(ctx, "auth.json", nil, apispec.ManagedFileMutationSequence(seq)); err == nil {
			t.Error("accepted invalid sequence")
		}
	}
	if _, err := client.Sandbox("sb_123").ManagedFiles(0, "").Inspect(ctx); err == nil {
		t.Error("accepted invalid generation")
	}
	if _, err := client.Sandbox("sb_123").ManagedFiles(7, "").Write(ctx, "auth.json", make([]byte, (4<<20)+1), "0"); err == nil {
		t.Error("accepted oversized content")
	}
}
