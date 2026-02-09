package sandbox0

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestHandleErrorResponse(t *testing.T) {
	envelope := apispec.ErrorEnvelope{
		Error: apispec.Error{Code: "bad_request", Message: "invalid"},
	}
	body, _ := envelope.MarshalJSON()
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}

	err := handleErrorResponse(nil, resp)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
	if apiErr.Code != "bad_request" || apiErr.Message != "invalid" {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
}

func TestAPIErrorFromHTTPResponseFallback(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
	}
	apiErr := apiErrorFromHTTPResponse(resp, []byte("boom"), false)
	if apiErr.Message != "boom" {
		t.Fatalf("unexpected message: %q", apiErr.Message)
	}
}

func TestRequestIDFromHeaders(t *testing.T) {
	headers := http.Header{"X-Request-Id": []string{"req-123"}}
	if got := requestIDFromHeaders(headers); got != "req-123" {
		t.Fatalf("unexpected request id: %s", got)
	}
}
