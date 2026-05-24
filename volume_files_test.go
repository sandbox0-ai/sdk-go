package sandbox0

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestUploadVolumeArchive(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/sandboxvolumes/vol-1/files/archive" {
			t.Fatalf("path = %s, want archive path", r.URL.Path)
		}
		if got := r.URL.Query().Get("path"); got != "/workspace" {
			t.Fatalf("path query = %q, want /workspace", got)
		}
		if got := r.URL.Query().Get("format"); got != "tar.gz" {
			t.Fatalf("format query = %q, want tar.gz", got)
		}
		if got := r.URL.Query().Get("overwrite"); got != "true" {
			t.Fatalf("overwrite query = %q, want true", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/gzip" {
			t.Fatalf("content type = %q, want application/gzip", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != "archive-data" {
			t.Fatalf("body = %q, want archive-data", string(body))
		}
		writeJSON(t, w, http.StatusCreated, map[string]any{
			"success": true,
			"data": map[string]any{
				"files":     3,
				"dirs":      1,
				"symlinks":  1,
				"bytes":     128,
				"overwrote": 2,
			},
		})
	})
	defer server.Close()

	result, err := client.UploadVolumeArchive(context.Background(), "vol-1", strings.NewReader("archive-data"), &VolumeArchiveUploadOptions{
		Path:      "/workspace",
		Format:    apispec.QueryArchiveFormatTarGz,
		Overwrite: true,
	})
	if err != nil {
		t.Fatalf("UploadVolumeArchive() error = %v", err)
	}
	if result.Files != 3 || result.Dirs != 1 || result.Symlinks != 1 || result.Bytes != 128 || result.Overwrote != 2 {
		t.Fatalf("result = %#v, want archive counts", result)
	}
}
