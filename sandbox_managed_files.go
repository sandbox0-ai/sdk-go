package sandbox0

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// ManagedFiles accesses node-owned runtime files without guest execution. It is
// bound to one generation and namespace (empty selects the projection root).
// All namespaces share the same platform mutation sequence. Native guest writes
// do not advance it. These methods never resume, discover a new generation or
// retry with a fresh sequence after an ambiguous response.
type ManagedFiles struct {
	client     *Client
	sandboxID  string
	generation int64
	namespace  string
}

func (s *Sandbox) ManagedFiles(runtimeGeneration int64, namespace string) *ManagedFiles {
	return &ManagedFiles{client: s.client, sandboxID: s.ID, generation: runtimeGeneration, namespace: namespace}
}

func validManagedFileName(name string) bool {
	if len(name) == 0 || len(name) > 128 || name == "." || name == ".." || strings.HasPrefix(name, ".s0-write-") {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func (f *ManagedFiles) validate() error {
	if f.generation <= 0 || (f.namespace != "" && !validManagedFileName(f.namespace)) {
		return errors.New("invalid managed file generation or namespace")
	}
	return nil
}

func (f *ManagedFiles) validateName(name string) error {
	if err := f.validate(); err != nil {
		return err
	}
	if !validManagedFileName(name) {
		return errors.New("invalid managed file name")
	}
	return nil
}

func validateManagedFileSequence(sequence apispec.ManagedFileMutationSequence) error {
	n, err := strconv.ParseUint(string(sequence), 10, 64)
	if err != nil || strconv.FormatUint(n, 10) != string(sequence) {
		return errors.New("invalid managed file mutation sequence")
	}
	return nil
}

// Inspect reports projection state, not historical credential safety of RootFS.
func (f *ManagedFiles) Inspect(ctx context.Context) (*apispec.ManagedFilesState, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}
	resp, err := f.client.api.InspectSandboxManagedFiles(ctx, apispec.InspectSandboxManagedFilesParams{ID: f.sandboxID, RuntimeGeneration: f.generation})
	if err != nil {
		return nil, err
	}
	return managedFilesState(resp, f.generation)
}

// Read returns decoded bytes and the current shared mutation sequence.
func (f *ManagedFiles) Read(ctx context.Context, name string) (*apispec.SuccessManagedFileReadResponseData, error) {
	if err := f.validateName(name); err != nil {
		return nil, err
	}
	var resp any
	var err error
	if f.namespace == "" {
		resp, err = f.client.api.ReadSandboxManagedFile(ctx, apispec.ReadSandboxManagedFileParams{ID: f.sandboxID, RuntimeGeneration: f.generation, Name: name})
	} else {
		resp, err = f.client.api.ReadSandboxNamespacedManagedFile(ctx, apispec.ReadSandboxNamespacedManagedFileParams{ID: f.sandboxID, RuntimeGeneration: f.generation, Namespace: f.namespace, Name: name})
	}
	if err != nil {
		return nil, err
	}
	if response, ok := resp.(*apispec.SuccessManagedFileReadResponse); ok {
		if response.Data.RuntimeGeneration != f.generation || len(response.Data.Content) > 4<<20 || validateManagedFileSequence(response.Data.MutationSequence) != nil {
			return nil, errors.New("invalid managed file response")
		}
		return &response.Data, nil
	}
	return nil, apiErrorFromResponse(resp)
}

// Write consumes expectedSequence before I/O. An error can mean the write was
// applied; inspect/read to reconcile before authorizing another mutation.
func (f *ManagedFiles) Write(ctx context.Context, name string, content []byte, expectedSequence apispec.ManagedFileMutationSequence) (*apispec.ManagedFilesState, error) {
	if err := f.validateName(name); err != nil {
		return nil, err
	}
	if err := validateManagedFileSequence(expectedSequence); err != nil {
		return nil, err
	}
	if len(content) > 4<<20 {
		return nil, errors.New("managed file exceeds size limit")
	}
	if content == nil {
		content = []byte{}
	}
	request := &apispec.ManagedFileWriteRequest{Content: content, ExpectedMutationSequence: expectedSequence}
	var resp any
	var err error
	if f.namespace == "" {
		resp, err = f.client.api.WriteSandboxManagedFile(ctx, request, apispec.WriteSandboxManagedFileParams{ID: f.sandboxID, RuntimeGeneration: f.generation, Name: name})
	} else {
		resp, err = f.client.api.WriteSandboxNamespacedManagedFile(ctx, request, apispec.WriteSandboxNamespacedManagedFileParams{ID: f.sandboxID, RuntimeGeneration: f.generation, Namespace: f.namespace, Name: name})
	}
	if err != nil {
		return nil, err
	}
	return managedFilesState(resp, f.generation)
}

// Delete consumes the sequence even if the file is already absent.
func (f *ManagedFiles) Delete(ctx context.Context, name string, expectedSequence apispec.ManagedFileMutationSequence) (*apispec.ManagedFilesState, error) {
	if err := f.validateName(name); err != nil {
		return nil, err
	}
	if err := validateManagedFileSequence(expectedSequence); err != nil {
		return nil, err
	}
	var resp any
	var err error
	if f.namespace == "" {
		resp, err = f.client.api.DeleteSandboxManagedFile(ctx, apispec.DeleteSandboxManagedFileParams{ID: f.sandboxID, RuntimeGeneration: f.generation, Name: name, ExpectedMutationSequence: expectedSequence})
	} else {
		resp, err = f.client.api.DeleteSandboxNamespacedManagedFile(ctx, apispec.DeleteSandboxNamespacedManagedFileParams{ID: f.sandboxID, RuntimeGeneration: f.generation, Namespace: f.namespace, Name: name, ExpectedMutationSequence: expectedSequence})
	}
	if err != nil {
		return nil, err
	}
	return managedFilesState(resp, f.generation)
}

func managedFilesState(resp any, generation int64) (*apispec.ManagedFilesState, error) {
	if response, ok := resp.(*apispec.SuccessManagedFilesStateResponse); ok {
		if response.Data.RuntimeGeneration != generation || validateManagedFileSequence(response.Data.MutationSequence) != nil {
			return nil, errors.New("invalid managed file response")
		}
		return &response.Data, nil
	}
	return nil, apiErrorFromResponse(resp)
}
