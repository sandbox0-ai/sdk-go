package sandbox0

import (
	"net/http"
	"strings"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestAPIErrorFormatting(t *testing.T) {
	var err *APIError
	if got := err.Error(); got != "<nil>" {
		t.Fatalf("expected <nil>, got %q", got)
	}
	err = &APIError{StatusCode: 400, Code: "bad_request", Message: "invalid"}
	if got := err.Error(); !strings.Contains(got, "bad_request") {
		t.Fatalf("unexpected error string: %q", got)
	}
}

func TestAPIErrorFromEnvelope(t *testing.T) {
	err := apiErrorFromEnvelope(0, nil)
	if err.StatusCode != 0 || err.Code != "unknown_error" {
		t.Fatalf("unexpected error: %#v", err)
	}

	envelope := &apispec.ErrorEnvelope{
		Error: apispec.Error{Code: "bad_request", Message: "invalid"},
	}
	err = apiErrorFromEnvelope(http.StatusBadRequest, envelope)
	if err.StatusCode != http.StatusBadRequest || err.Code != "bad_request" {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestUnexpectedResponseError(t *testing.T) {
	err := unexpectedResponseError(nil)
	if err.StatusCode != 0 || err.Code != "unexpected_response" {
		t.Fatalf("unexpected error: %#v", err)
	}

	notFound := apispec.APIV1SandboxesIDDeleteNotFound(apispec.ErrorEnvelope{
		Error: apispec.Error{Code: "not_found", Message: "nope"},
	})
	err = unexpectedResponseError(&notFound)
	if err.Code != "unexpected_response" || err.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected error: %#v", err)
	}
}
