package sandbox0

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

var finalCaptureIDPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func validateFinalCaptureTarget(sandboxID, operationID string) error {
	if sandboxID == "" || strings.TrimSpace(sandboxID) != sandboxID || sandboxID == "." || sandboxID == ".." || !restartIDPattern.MatchString(operationID) {
		return errors.New("invalid restart final capture identity")
	}
	return nil
}

// GetSandboxRestartFinalCapture reads durable stopped-source custody metadata.
func (c *Client) GetSandboxRestartFinalCapture(ctx context.Context, sandboxID, operationID string) (*apispec.SandboxRestartFinalCapture, error) {
	if err := validateFinalCaptureTarget(sandboxID, operationID); err != nil {
		return nil, err
	}
	response, err := c.api.GetSandboxRestartFinalCapture(ctx, apispec.GetSandboxRestartFinalCaptureParams{ID: sandboxID, OperationID: operationID})
	if err != nil {
		return nil, err
	}
	return finalCaptureStatus(response, sandboxID, operationID)
}

// ReadSandboxRestartFinalCaptureFile reads transient bytes from the stopped
// source. Only Found=false is authoritative absence; errors never imply logout.
// Encrypt and commit owner state before acknowledging custody release.
func (c *Client) ReadSandboxRestartFinalCaptureFile(ctx context.Context, sandboxID, operationID, captureID, namespace, name string) (*apispec.SandboxRestartFinalCaptureFile, error) {
	if err := validateFinalCaptureTarget(sandboxID, operationID); err != nil {
		return nil, err
	}
	if !finalCaptureIDPattern.MatchString(captureID) || !validManagedFileName(namespace) || !validManagedFileName(name) {
		return nil, errors.New("invalid final capture file identity")
	}
	response, err := c.api.ReadSandboxRestartFinalCaptureFile(ctx, apispec.ReadSandboxRestartFinalCaptureFileParams{ID: sandboxID, OperationID: operationID, CaptureID: captureID, Namespace: namespace, Name: name})
	if err != nil {
		return nil, err
	}
	v, ok := response.(*apispec.SuccessSandboxRestartFinalCaptureFileResponse)
	if !ok {
		return nil, apiErrorFromResponse(response)
	}
	r := &v.Data
	if r.OperationID != operationID || r.CaptureID != captureID || r.RuntimeGeneration < 1 || r.SizeBytes < 1<<20 || r.SizeBytes > 64<<20 || len(r.Content) > 4<<20 || (!r.Found && len(r.Content) != 0) || validateManagedFileSequence(apispec.ManagedFileMutationSequence(r.MutationSequence)) != nil {
		return nil, errors.New("invalid final capture file response")
	}
	return r, nil
}

// AcknowledgeSandboxRestartFinalCapture authorizes custody release only after
// the owner durably commits encrypted credentials or logout. Reconcile uncertain
// responses using the same capture and receipt IDs; this method never retries.
func (c *Client) AcknowledgeSandboxRestartFinalCapture(ctx context.Context, sandboxID, operationID, captureID, receiptID string) (*apispec.SandboxRestartFinalCapture, error) {
	if err := validateFinalCaptureTarget(sandboxID, operationID); err != nil {
		return nil, err
	}
	if !finalCaptureIDPattern.MatchString(captureID) || !restartKeyPattern.MatchString(receiptID) {
		return nil, errors.New("invalid final capture acknowledgement")
	}
	response, err := c.api.AcknowledgeSandboxRestartFinalCapture(ctx, &apispec.SandboxRestartFinalCaptureAcknowledgement{CaptureID: captureID, ReceiptID: receiptID}, apispec.AcknowledgeSandboxRestartFinalCaptureParams{ID: sandboxID, OperationID: operationID})
	if err != nil {
		return nil, err
	}
	r, err := finalCaptureStatus(response, sandboxID, operationID)
	if err != nil {
		return nil, err
	}
	if r.CaptureID.Value != captureID || r.ReceiptID.Value != receiptID || (r.State != "acknowledged" && r.State != "released") {
		return nil, errors.New("final capture acknowledgement changed receipt")
	}
	return r, nil
}

func finalCaptureStatus(response any, sandboxID, operationID string) (*apispec.SandboxRestartFinalCapture, error) {
	v, ok := response.(*apispec.SuccessSandboxRestartFinalCaptureResponse)
	if !ok {
		return nil, apiErrorFromResponse(response)
	}
	r := &v.Data
	if r.SandboxID != sandboxID || r.OperationID != operationID || r.FromRuntimeGeneration < 1 {
		return nil, errors.New("invalid final capture status")
	}
	switch r.State {
	case "pending":
		if r.CaptureID.IsSet() || r.StoppedAt.IsSet() || r.ReceiptID.IsSet() {
			return nil, errors.New("pending final capture has a receipt")
		}
	case "stopped", "acknowledged", "released":
		if !r.CaptureID.IsSet() || !finalCaptureIDPattern.MatchString(r.CaptureID.Value) || !r.StoppedAt.IsSet() || r.StoppedAt.Value.IsZero() {
			return nil, errors.New("invalid stopped capture identity")
		}
		if r.State == "stopped" {
			if r.ReceiptID.IsSet() {
				return nil, errors.New("unacknowledged capture has a receipt")
			}
		} else if !r.ReceiptID.IsSet() || !restartKeyPattern.MatchString(r.ReceiptID.Value) {
			return nil, errors.New("invalid final capture receipt")
		}
	default:
		return nil, errors.New("invalid final capture state")
	}
	return r, nil
}
