package sandbox0

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

const maxErrorBodyBytes = 64 * 1024

func handleErrorResponse(_ context.Context, resp *http.Response) error {
	if resp == nil || resp.Body == nil {
		return nil
	}
	if resp.StatusCode < http.StatusBadRequest {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes+1))
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	truncated := false
	if len(body) > maxErrorBodyBytes {
		body = body[:maxErrorBodyBytes]
		truncated = true
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))

	return apiErrorFromHTTPResponse(resp, body, truncated)
}

func apiErrorFromHTTPResponse(resp *http.Response, body []byte, truncated bool) *APIError {
	err := &APIError{
		StatusCode: resp.StatusCode,
		RequestID:  requestIDFromHeaders(resp.Header),
		Body:       body,
	}

	if isJSONContentType(resp.Header.Get("Content-Type")) && len(body) > 0 {
		var envelope apispec.ErrorEnvelope
		if json.Unmarshal(body, &envelope) == nil && envelope.Error.Message != "" {
			err.Code = envelope.Error.Code
			err.Message = envelope.Error.Message
			err.Details = envelope.Error.Details
			return err
		}
		if compacted := compactJSONString(body); compacted != "" {
			err.Message = compacted
			if truncated {
				err.Message += " (truncated)"
			}
			return err
		}
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}
	if truncated && message != "" {
		message += " (truncated)"
	}
	err.Message = message
	return err
}

func requestIDFromHeaders(headers http.Header) string {
	for _, key := range []string{
		"X-Request-Id",
		"X-Request-ID",
		"Request-Id",
		"Request-ID",
	} {
		if value := strings.TrimSpace(headers.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func isJSONContentType(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "application/json") ||
		strings.Contains(value, "application/problem+json")
}

func compactJSONString(body []byte) string {
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, body); err != nil {
		return ""
	}
	return compacted.String()
}
