package sandbox0

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestTeamQuotaPolicyWriteRequestVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		request  apispec.TeamQuotaPolicyWriteRequest
		expected map[string]any
	}{
		{
			name: "capacity",
			request: apispec.NewTeamQuotaCapacityPolicyWriteRequestTeamQuotaPolicyWriteRequest(
				apispec.TeamQuotaCapacityPolicyWriteRequest{
					Kind:  apispec.TeamQuotaCapacityPolicyWriteRequestKindCapacity,
					Limit: 100,
				},
			),
			expected: map[string]any{"kind": "capacity", "limit": float64(100)},
		},
		{
			name: "concurrency",
			request: apispec.NewTeamQuotaConcurrencyPolicyWriteRequestTeamQuotaPolicyWriteRequest(
				apispec.TeamQuotaConcurrencyPolicyWriteRequest{
					Kind:  apispec.TeamQuotaConcurrencyPolicyWriteRequestKindConcurrency,
					Limit: 25,
				},
			),
			expected: map[string]any{"kind": "concurrency", "limit": float64(25)},
		},
		{
			name: "rate",
			request: apispec.NewTeamQuotaRatePolicyWriteRequestTeamQuotaPolicyWriteRequest(
				apispec.TeamQuotaRatePolicyWriteRequest{
					Kind:       apispec.TeamQuotaRatePolicyWriteRequestKindRate,
					Tokens:     20,
					IntervalMs: 1000,
					Burst:      40,
				},
			),
			expected: map[string]any{
				"kind":        "rate",
				"tokens":      float64(20),
				"interval_ms": float64(1000),
				"burst":       float64(40),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(payload, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if !reflect.DeepEqual(decoded, tt.expected) {
				t.Fatalf("encoded policy = %#v, want %#v", decoded, tt.expected)
			}
		})
	}
}

func TestTeamQuotaPolicyWriteRequestDecodesByKind(t *testing.T) {
	t.Parallel()

	var request apispec.TeamQuotaPolicyWriteRequest
	if err := json.Unmarshal([]byte(`{"kind":"concurrency","limit":25}`), &request); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	policy, ok := request.GetTeamQuotaConcurrencyPolicyWriteRequest()
	if !ok {
		t.Fatalf("decoded variant = %q, want concurrency", request.Type)
	}
	if policy.Kind != apispec.TeamQuotaConcurrencyPolicyWriteRequestKindConcurrency || policy.Limit != 25 {
		t.Fatalf("decoded policy = %+v", policy)
	}
}
