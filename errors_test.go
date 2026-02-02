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
	err := apiErrorFromEnvelope(nil, nil)
	if err.StatusCode != 0 || err.Code != "unknown_error" {
		t.Fatalf("unexpected error: %#v", err)
	}

	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Status:     "400 Bad Request",
		Header:     http.Header{"X-Request-Id": []string{"req-1"}},
	}
	envelope := &apispec.ErrorEnvelope{
		Error: apispec.Error{Code: "bad_request", Message: "invalid"},
	}
	err = apiErrorFromEnvelope(resp, envelope)
	if err.RequestID != "req-1" || err.Code != "bad_request" {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestUnexpectedResponseError(t *testing.T) {
	err := unexpectedResponseError(nil, []byte("x"))
	if err.StatusCode != 0 || err.Code != "unexpected_response" {
		t.Fatalf("unexpected error: %#v", err)
	}

	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Status:     "403 Forbidden",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	body := []byte(`{"success":false,"error":{"code":"forbidden","message":"nope"}}`)
	err = unexpectedResponseError(resp, body)
	if err.Code != "forbidden" || err.StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected error: %#v", err)
	}

	resp = &http.Response{
		StatusCode: http.StatusInternalServerError,
		Status:     "500 Internal Server Error",
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
	}
	err = unexpectedResponseError(resp, []byte("boom"))
	if err.Code != "unexpected_response" || err.StatusCode != http.StatusInternalServerError {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestRequestIDFromResponse(t *testing.T) {
	if got := requestIDFromResponse(nil); got != "" {
		t.Fatalf("expected empty request id, got %q", got)
	}
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("X-Request-Id", "req-a")
	if got := requestIDFromResponse(resp); got != "req-a" {
		t.Fatalf("unexpected request id: %q", got)
	}
	resp = &http.Response{Header: http.Header{}}
	resp.Header.Set("X-Request-ID", "req-b")
	if got := requestIDFromResponse(resp); got != "req-b" {
		t.Fatalf("unexpected request id: %q", got)
	}
}
