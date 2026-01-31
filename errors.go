package sandbox0

import (
	"fmt"
	"net/http"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// APIError represents a structured error returned by the Sandbox0 API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
	Details    interface{}
	Body       []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Code != "" {
		return fmt.Sprintf("sandbox0 API error (%d): %s - %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("sandbox0 API error (%d): %s", e.StatusCode, e.Message)
}

func apiErrorFromEnvelope(resp *http.Response, envelope *apispec.ErrorEnvelope) *APIError {
	if resp == nil {
		return &APIError{
			StatusCode: 0,
			Code:       "unknown_error",
			Message:    "no response received",
		}
	}
	requestID := requestIDFromResponse(resp)
	if envelope == nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Code:       "unknown_error",
			Message:    resp.Status,
			RequestID:  requestID,
		}
	}
	return &APIError{
		StatusCode: resp.StatusCode,
		Code:       envelope.Error.Code,
		Message:    envelope.Error.Message,
		Details:    envelope.Error.Details,
		RequestID:  requestID,
	}
}

func unexpectedResponseError(resp *http.Response, body []byte) *APIError {
	if resp == nil {
		return &APIError{
			StatusCode: 0,
			Code:       "unexpected_response",
			Message:    "no response received",
			Body:       body,
		}
	}
	return &APIError{
		StatusCode: resp.StatusCode,
		Code:       "unexpected_response",
		Message:    resp.Status,
		RequestID:  requestIDFromResponse(resp),
		Body:       body,
	}
}

func requestIDFromResponse(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	if id := resp.Header.Get("X-Request-Id"); id != "" {
		return id
	}
	if id := resp.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return ""
}
