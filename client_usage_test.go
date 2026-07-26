package sandbox0

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestListUsageWindows(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/usage/windows" {
			t.Fatalf("path = %s, want /api/v1/usage/windows", r.URL.Path)
		}
		query := r.URL.Query()
		if got := query.Get("cursor"); got != "page-1" {
			t.Fatalf("cursor = %q, want page-1", got)
		}
		if got := query.Get("limit"); got != "250" {
			t.Fatalf("limit = %q, want 250", got)
		}
		if got := query.Get("window_type"); got != "sandbox.runtime_mib_milliseconds" {
			t.Fatalf("window_type = %q, want sandbox.runtime_mib_milliseconds", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}

		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"windows": []map[string]any{
					{
						"window_id":    "window-1",
						"window_type":  "sandbox.runtime_mib_milliseconds",
						"subject_type": "sandbox",
						"subject_id":   "sandbox-1",
						"sandbox_id":   "sandbox-1",
						"window_start": "2026-07-26T00:00:00Z",
						"window_end":   "2026-07-26T01:00:00Z",
						"value":        3_686_400_000,
						"unit":         "mib_milliseconds",
						"recorded_at":  "2026-07-26T01:00:01Z",
					},
				},
				"next_cursor": "page-2",
			},
		})
	})
	defer server.Close()

	page, err := client.ListUsageWindows(context.Background(), &ListUsageWindowsOptions{
		Cursor:     "page-1",
		Limit:      250,
		WindowType: "sandbox.runtime_mib_milliseconds",
	})
	if err != nil {
		t.Fatalf("ListUsageWindows() error = %v", err)
	}
	if page.NextCursor != "page-2" {
		t.Fatalf("NextCursor = %q, want page-2", page.NextCursor)
	}
	if len(page.Windows) != 1 {
		t.Fatalf("len(Windows) = %d, want 1", len(page.Windows))
	}
	window := page.Windows[0]
	if window.WindowID != "window-1" {
		t.Fatalf("WindowID = %q, want window-1", window.WindowID)
	}
	if window.Value != 3_686_400_000 {
		t.Fatalf("Value = %d, want 3686400000", window.Value)
	}
	if sandboxID, ok := window.SandboxID.Get(); !ok || sandboxID != "sandbox-1" {
		t.Fatalf("SandboxID = %q, %v, want sandbox-1, true", sandboxID, ok)
	}
	wantStart := time.Date(2026, time.July, 26, 0, 0, 0, 0, time.UTC)
	if !window.WindowStart.Equal(wantStart) {
		t.Fatalf("WindowStart = %s, want %s", window.WindowStart, wantStart)
	}
}
