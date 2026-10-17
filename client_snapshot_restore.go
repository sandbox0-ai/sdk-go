package sandbox0

import (
	"context"
	"errors"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// RestoreSandboxSnapshot replaces the writable RootFS with an exact snapshot and
// explicitly ends processes. Keep every request field and key after uncertainty.
// Acceptance is asynchronous; inspect the returned operation until terminal.
// This never discovers a fresh generation or falls back from memory restore.
func (c *Client) RestoreSandboxSnapshot(ctx context.Context, sandboxID string, request apispec.SandboxSnapshotRestoreRequest, idempotencyKey string) (*apispec.SandboxSnapshotRestoreOperation, error) {
	if request.ExpectedRuntimeGeneration < 1 || request.ExpectedRuntimeGeneration >= 9007199254740991 || !bool(request.DiscardProcessState) || !snapshotRestoreIdentity(request.SnapshotID) || !snapshotRestoreIdentity(request.SnapshotGenerationID) || !restartKeyPattern.MatchString(idempotencyKey) {
		return nil, errors.New("invalid snapshot restore request")
	}
	response, err := c.api.RestoreSandboxSnapshot(ctx, &request, apispec.RestoreSandboxSnapshotParams{ID: sandboxID, IdempotencyKey: idempotencyKey})
	if err != nil {
		return nil, err
	}
	r, err := snapshotRestoreOperation(response, sandboxID)
	if err != nil {
		return nil, err
	}
	if r.RequestKey != idempotencyKey || r.FromRuntimeGeneration != request.ExpectedRuntimeGeneration || r.SnapshotID != request.SnapshotID || r.SnapshotGenerationID != request.SnapshotGenerationID {
		return nil, errors.New("snapshot restore receipt does not match request")
	}
	return r, nil
}

// GetSandboxSnapshotRestoreOperation reads a historical receipt without guest execution.
func (c *Client) GetSandboxSnapshotRestoreOperation(ctx context.Context, sandboxID, operationID string) (*apispec.SandboxSnapshotRestoreOperation, error) {
	if !restartIDPattern.MatchString(operationID) {
		return nil, errors.New("invalid snapshot restore operation ID")
	}
	response, err := c.api.GetSandboxSnapshotRestoreOperation(ctx, apispec.GetSandboxSnapshotRestoreOperationParams{ID: sandboxID, OperationID: operationID})
	if err != nil {
		return nil, err
	}
	r, err := snapshotRestoreOperation(response, sandboxID)
	if err != nil {
		return nil, err
	}
	if r.ID != operationID {
		return nil, errors.New("snapshot restore receipt does not match operation")
	}
	return r, nil
}

// FindSandboxSnapshotRestoreOperation recovers a lost submission acknowledgement. A
// missing receipt does not rule out a concurrent submission committing later.
func (c *Client) FindSandboxSnapshotRestoreOperation(ctx context.Context, sandboxID, requestKey string) (*apispec.SandboxSnapshotRestoreOperation, error) {
	if !restartKeyPattern.MatchString(requestKey) {
		return nil, errors.New("invalid restart request key")
	}
	response, err := c.api.FindSandboxSnapshotRestoreOperation(ctx, apispec.FindSandboxSnapshotRestoreOperationParams{ID: sandboxID, RequestKey: requestKey})
	if err != nil {
		return nil, err
	}
	r, err := snapshotRestoreOperation(response, sandboxID)
	if err != nil {
		return nil, err
	}
	if r.RequestKey != requestKey {
		return nil, errors.New("snapshot restore receipt does not match request key")
	}
	return r, nil
}

func snapshotRestoreOperation(response any, sandboxID string) (*apispec.SandboxSnapshotRestoreOperation, error) {
	var r apispec.SandboxSnapshotRestoreOperation
	switch v := response.(type) {
	case *apispec.RestoreSandboxSnapshotAccepted:
		r = (*apispec.SuccessSandboxSnapshotRestoreOperationResponse)(v).Data
	case *apispec.RestoreSandboxSnapshotOK:
		r = (*apispec.SuccessSandboxSnapshotRestoreOperationResponse)(v).Data
	case *apispec.SuccessSandboxSnapshotRestoreOperationResponse:
		r = v.Data
	default:
		return nil, apiErrorFromResponse(response)
	}
	if r.SandboxID != sandboxID || !restartIDPattern.MatchString(r.ID) || !restartKeyPattern.MatchString(r.RequestKey) || r.FromRuntimeGeneration < 1 || r.FromRuntimeGeneration >= 9007199254740991 || !bool(r.DiscardProcessState) || !snapshotRestoreIdentity(r.SnapshotID) || !snapshotRestoreIdentity(r.SnapshotGenerationID) {
		return nil, errors.New("invalid snapshot restore receipt")
	}
	generation, set := r.ResultRuntimeGeneration.Get()
	if string(r.State) == "succeeded" {
		if !set || generation != r.FromRuntimeGeneration+1 {
			return nil, errors.New("invalid snapshot restore result generation")
		}
	} else if set {
		return nil, errors.New("uncommitted snapshot restore result generation")
	}
	if (string(r.State) == "resuming" || string(r.State) == "succeeded") && !r.RestoredRootfsGenerationID.Set {
		return nil, errors.New("restored head receipt missing")
	}
	if r.RestoredRootfsGenerationID.Set && !snapshotRestoreIdentity(r.RestoredRootfsGenerationID.Value) {
		return nil, errors.New("invalid restored head")
	}
	return &r, nil
}

func snapshotRestoreIdentity(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && len(value) <= 512 && !strings.ContainsRune(value, 0)
}

// GetSandboxSnapshotRestoreFinalCapture reads durable stopped-source custody metadata.
func (c *Client) GetSandboxSnapshotRestoreFinalCapture(ctx context.Context, sandboxID, operationID string) (*apispec.SandboxRestartFinalCapture, error) {
	if err := validateFinalCaptureTarget(sandboxID, operationID); err != nil {
		return nil, err
	}
	response, err := c.api.GetSandboxSnapshotRestoreFinalCapture(ctx, apispec.GetSandboxSnapshotRestoreFinalCaptureParams{ID: sandboxID, OperationID: operationID})
	if err != nil {
		return nil, err
	}
	return finalCaptureStatus(response, sandboxID, operationID)
}

// ReadSandboxSnapshotRestoreFinalCaptureFile reads transient bytes from the stopped
// source. Only Found=false is authoritative absence; errors never imply logout.
// Encrypt and commit owner state before acknowledging custody release.
func (c *Client) ReadSandboxSnapshotRestoreFinalCaptureFile(ctx context.Context, sandboxID, operationID, captureID, namespace, name string) (*apispec.SandboxRestartFinalCaptureFile, error) {
	if err := validateFinalCaptureTarget(sandboxID, operationID); err != nil {
		return nil, err
	}
	if !finalCaptureIDPattern.MatchString(captureID) || !validManagedFileName(namespace) || !validManagedFileName(name) {
		return nil, errors.New("invalid final capture file identity")
	}
	response, err := c.api.ReadSandboxSnapshotRestoreFinalCaptureFile(ctx, apispec.ReadSandboxSnapshotRestoreFinalCaptureFileParams{ID: sandboxID, OperationID: operationID, CaptureID: captureID, Namespace: namespace, Name: name})
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

// AcknowledgeSandboxSnapshotRestoreFinalCapture authorizes custody release only after
// the owner durably commits encrypted credentials or logout. Reconcile uncertain
// responses using the same capture and receipt IDs; this method never retries.
func (c *Client) AcknowledgeSandboxSnapshotRestoreFinalCapture(ctx context.Context, sandboxID, operationID, captureID, receiptID string) (*apispec.SandboxRestartFinalCapture, error) {
	if err := validateFinalCaptureTarget(sandboxID, operationID); err != nil {
		return nil, err
	}
	if !finalCaptureIDPattern.MatchString(captureID) || !restartKeyPattern.MatchString(receiptID) {
		return nil, errors.New("invalid final capture acknowledgement")
	}
	response, err := c.api.AcknowledgeSandboxSnapshotRestoreFinalCapture(ctx, &apispec.SandboxRestartFinalCaptureAcknowledgement{CaptureID: captureID, ReceiptID: receiptID}, apispec.AcknowledgeSandboxSnapshotRestoreFinalCaptureParams{ID: sandboxID, OperationID: operationID})
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
