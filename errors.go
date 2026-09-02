package sandbox0

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

const CodeClaimStartThrottled = "claim_start_throttled"

// SandboxWaitTimeoutError reports that a sandbox did not reach the requested
// committed lifecycle state before the wait deadline.
type SandboxWaitTimeoutError struct {
	SandboxID   string
	Timeout     time.Duration
	LastSandbox *apispec.Sandbox
}

func (e *SandboxWaitTimeoutError) Error() string {
	return fmt.Sprintf("timed out waiting for sandbox %s after %s", e.SandboxID, e.Timeout)
}

// APIError represents a structured error returned by the Sandbox0 API.
type APIError struct {
	StatusCode        int
	Code              string
	Message           string
	RequestID         string
	Details           any
	Body              []byte
	RetryAfterSeconds int
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

func (e *APIError) IsClaimStartThrottled() bool {
	return e != nil &&
		e.StatusCode == http.StatusTooManyRequests &&
		e.Code == CodeClaimStartThrottled
}

func IsClaimStartThrottled(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.IsClaimStartThrottled()
}

func apiErrorFromEnvelope(statusCode int, envelope *apispec.ErrorEnvelope, retryAfterSeconds int) *APIError {
	if envelope == nil {
		return &APIError{
			StatusCode:        statusCode,
			Code:              "unknown_error",
			Message:           "no error body received",
			RetryAfterSeconds: retryAfterSeconds,
		}
	}
	return &APIError{
		StatusCode:        statusCode,
		Code:              envelope.Error.Code,
		Message:           envelope.Error.Message,
		Details:           envelope.Error.Details,
		RetryAfterSeconds: retryAfterSeconds,
	}
}

func apiErrorFromResponse(res any) *APIError {
	status := errorStatusFromResponse(res)
	envelope, retryAfterSeconds, ok := errorEnvelopeFromResponse(res)
	if ok {
		return apiErrorFromEnvelope(status, envelope, retryAfterSeconds)
	}
	return &APIError{
		StatusCode: status,
		Code:       "unexpected_response",
		Message:    "unexpected response",
	}
}

func errorEnvelopeFromResponse(res any) (*apispec.ErrorEnvelope, int, bool) {
	if res == nil {
		return nil, 0, false
	}
	if envelope, ok := res.(*apispec.ErrorEnvelope); ok {
		return envelope, 0, true
	}
	value := reflect.ValueOf(res)
	if value.Kind() != reflect.Pointer {
		return nil, 0, false
	}
	if envelope, retryAfterSeconds, ok := errorEnvelopeHeadersFromValue(value); ok {
		return envelope, retryAfterSeconds, true
	}
	target := reflect.TypeOf(&apispec.ErrorEnvelope{})
	if !value.Type().ConvertibleTo(target) {
		return nil, 0, false
	}
	converted := value.Convert(target)
	envelope, ok := converted.Interface().(*apispec.ErrorEnvelope)
	return envelope, 0, ok
}

func errorEnvelopeHeadersFromValue(value reflect.Value) (*apispec.ErrorEnvelope, int, bool) {
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return nil, 0, false
	}
	target := reflect.TypeOf(&apispec.ErrorEnvelopeHeaders{})
	if value.Type().ConvertibleTo(target) {
		headers := value.Convert(target).Interface().(*apispec.ErrorEnvelopeHeaders)
		return &headers.Response, headers.RetryAfter.Or(0), true
	}
	elem := value.Elem()
	if elem.Kind() != reflect.Struct {
		return nil, 0, false
	}
	responseField := elem.FieldByName("Response")
	if !responseField.IsValid() || !responseField.CanInterface() {
		return nil, 0, false
	}
	envelope, ok := responseField.Interface().(apispec.ErrorEnvelope)
	if !ok {
		return nil, 0, false
	}
	retryAfterSeconds := 0
	retryAfterField := elem.FieldByName("RetryAfter")
	if retryAfterField.IsValid() && retryAfterField.CanInterface() {
		if retryAfter, ok := retryAfterField.Interface().(apispec.OptInt); ok {
			retryAfterSeconds = retryAfter.Or(0)
		}
	}
	return &envelope, retryAfterSeconds, true
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

func expectResponse[T any](res any) (*T, error) {
	response, ok := res.(*T)
	if !ok {
		return nil, apiErrorFromResponse(res)
	}
	return response, nil
}
