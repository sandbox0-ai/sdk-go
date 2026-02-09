//go:build e2e

package sandbox0_test

import (
	"strings"
	"testing"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
)

func TestAPIErrorString(t *testing.T) {
	err := (&sandbox0.APIError{
		StatusCode: 400,
		Code:       "bad_request",
		Message:    "invalid input",
	}).Error()
	if !strings.Contains(err, "bad_request") {
		t.Fatalf("expected error string to include code, got %q", err)
	}
}
