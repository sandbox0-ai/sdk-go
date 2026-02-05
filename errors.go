package sandbox0

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// APIError represents a structured error returned by the Sandbox0 API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
	Details    any
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

func apiErrorFromEnvelope(statusCode int, envelope *apispec.ErrorEnvelope) *APIError {
	if envelope == nil {
		return &APIError{
			StatusCode: statusCode,
			Code:       "unknown_error",
			Message:    "no error body received",
		}
	}
	return &APIError{
		StatusCode: statusCode,
		Code:       envelope.Error.Code,
		Message:    envelope.Error.Message,
		Details:    envelope.Error.Details,
	}
}

func apiErrorFromResponse(res any) *APIError {
	status := errorStatusFromResponse(res)
	envelope, ok := errorEnvelopeFromResponse(res)
	if ok {
		return apiErrorFromEnvelope(status, envelope)
	}
	return &APIError{
		StatusCode: status,
		Code:       "unexpected_response",
		Message:    "unexpected response",
	}
}

func errorEnvelopeFromResponse(res any) (*apispec.ErrorEnvelope, bool) {
	if res == nil {
		return nil, false
	}
	if envelope, ok := res.(*apispec.ErrorEnvelope); ok {
		return envelope, true
	}
	value := reflect.ValueOf(res)
	if value.Kind() != reflect.Pointer {
		return nil, false
	}
	target := reflect.TypeOf(&apispec.ErrorEnvelope{})
	if !value.Type().ConvertibleTo(target) {
		return nil, false
	}
	converted := value.Convert(target)
	envelope, ok := converted.Interface().(*apispec.ErrorEnvelope)
	return envelope, ok
}

func errorStatusFromResponse(res any) int {
	if res == nil {
		return 0
	}
	t := reflect.TypeOf(res)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	name := t.Name()
	switch {
	case strings.HasSuffix(name, "BadRequest"):
		return http.StatusBadRequest
	case strings.HasSuffix(name, "Unauthorized"):
		return http.StatusUnauthorized
	case strings.HasSuffix(name, "Forbidden"):
		return http.StatusForbidden
	case strings.HasSuffix(name, "NotFound"):
		return http.StatusNotFound
	case strings.HasSuffix(name, "Conflict"):
		return http.StatusConflict
	case strings.HasSuffix(name, "TooManyRequests"):
		return http.StatusTooManyRequests
	case strings.HasSuffix(name, "InternalServerError"):
		return http.StatusInternalServerError
	case strings.HasSuffix(name, "ServiceUnavailable"):
		return http.StatusServiceUnavailable
	default:
		return 0
	}
}

func unexpectedResponseError(res any) *APIError {
	if res == nil {
		return &APIError{
			StatusCode: 0,
			Code:       "unexpected_response",
			Message:    "no response received",
		}
	}
	return &APIError{
		StatusCode: errorStatusFromResponse(res),
		Code:       "unexpected_response",
		Message:    "unexpected response",
	}
}
