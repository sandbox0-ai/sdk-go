package sandbox0

import (
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestAPIErrorFromAdmissionResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		response   any
		statusCode int
		code       string
		retryAfter int
	}{
		{
			name: "rate limited",
			response: &apispec.AdmissionRateLimitedHeaders{
				RetryAfter: apispec.NewOptInt(2),
				Response: apispec.ErrorEnvelope{
					Error: apispec.Error{Code: "api_requests", Message: "rate limited"},
				},
			},
			statusCode: http.StatusTooManyRequests,
			code:       "api_requests",
			retryAfter: 2,
		},
		{
			name: "unavailable",
			response: &apispec.AdmissionUnavailableHeaders{
				RetryAfter: apispec.NewOptInt(3),
				Response: apispec.ErrorEnvelope{
					Error: apispec.Error{Code: "quota_unavailable", Message: "unavailable"},
				},
			},
			statusCode: http.StatusServiceUnavailable,
			code:       "quota_unavailable",
			retryAfter: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := apiErrorFromResponse(tt.response)
			if err.StatusCode != tt.statusCode || err.Code != tt.code || err.RetryAfterSeconds != tt.retryAfter {
				t.Fatalf("apiErrorFromResponse() = %+v", err)
			}
		})
	}
}
