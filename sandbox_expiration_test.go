package sandbox0

import (
	"encoding/json"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxSummaryAcceptsDisabledExpirations(t *testing.T) {
	const payload = `{
		"id":"sb_disabled_ttl",
		"template_id":"default",
		"status":"running",
		"paused":false,
		"runtime_generation":1,
		"created_at":"2026-08-03T00:00:00Z",
		"expires_at":null,
		"hard_expires_at":null,
		"updated_at":"2026-08-03T00:00:00Z"
	}`

	var summary apispec.SandboxSummary
	if err := json.Unmarshal([]byte(payload), &summary); err != nil {
		t.Fatalf("decode sandbox summary: %v", err)
	}
	if !summary.ExpiresAt.IsSet() || !summary.ExpiresAt.IsNull() {
		t.Fatalf("expires_at = %+v, want explicit null", summary.ExpiresAt)
	}
	if !summary.HardExpiresAt.IsSet() || !summary.HardExpiresAt.IsNull() {
		t.Fatalf("hard_expires_at = %+v, want explicit null", summary.HardExpiresAt)
	}
}
