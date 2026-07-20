package sandbox0

import (
	"context"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestListTeamQuotas(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/quotas" {
			t.Fatalf("path = %s, want /api/v1/quotas", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data": []map[string]any{
				{
					"team_id":     "team-1",
					"dimension":   "active_sandboxes",
					"kind":        "capacity",
					"limit_value": 10,
					"interval_ms": nil,
					"burst_value": nil,
					"current":     3,
					"remaining":   7,
					"unlimited":   false,
					"unit":        "count",
					"source":      "region_default",
				},
				{
					"team_id":     "team-1",
					"dimension":   "api_requests",
					"kind":        "rate",
					"limit_value": 100,
					"interval_ms": 1000,
					"burst_value": 200,
					"current":     nil,
					"remaining":   nil,
					"unlimited":   false,
					"unit":        "requests",
					"source":      "team_override",
				},
			},
		})
	})
	defer server.Close()

	quotas, err := client.ListTeamQuotas(context.Background())
	if err != nil {
		t.Fatalf("ListTeamQuotas() error = %v", err)
	}
	if len(quotas) != 2 {
		t.Fatalf("len(quotas) = %d, want 2", len(quotas))
	}
	if got := quotas[0].Dimension; got != apispec.QuotaDimensionActiveSandboxes {
		t.Fatalf("first dimension = %q, want active_sandboxes", got)
	}
	if current, ok := quotas[0].Current.Get(); !ok || current != 3 {
		t.Fatalf("first current = %d, %v, want 3, true", current, ok)
	}
	if got := quotas[1].Kind; got != apispec.TeamQuotaKindRate {
		t.Fatalf("second kind = %q, want rate", got)
	}
	if !quotas[1].Current.IsNull() || !quotas[1].Remaining.IsNull() {
		t.Fatal("rate quota current and remaining should be null")
	}
}

func TestGetTeamQuota(t *testing.T) {
	client, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/quotas/network_egress_bytes" {
			t.Fatalf("path = %s, want /api/v1/quotas/network_egress_bytes", r.URL.Path)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"team_id":     "team-1",
				"dimension":   "network_egress_bytes",
				"kind":        "rate",
				"limit_value": 1048576,
				"interval_ms": 1000,
				"burst_value": 2097152,
				"current":     nil,
				"remaining":   nil,
				"unlimited":   false,
				"unit":        "bytes",
				"source":      "region_default",
			},
		})
	})
	defer server.Close()

	quota, err := client.GetTeamQuota(context.Background(), apispec.QuotaDimensionNetworkEgressBytes)
	if err != nil {
		t.Fatalf("GetTeamQuota() error = %v", err)
	}
	if got := quota.Dimension; got != apispec.QuotaDimensionNetworkEgressBytes {
		t.Fatalf("dimension = %q, want network_egress_bytes", got)
	}
	if limit, ok := quota.LimitValue.Get(); !ok || limit != 1048576 {
		t.Fatalf("limit = %d, %v, want 1048576, true", limit, ok)
	}
}
