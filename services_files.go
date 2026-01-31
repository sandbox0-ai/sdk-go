package sandbox0

import (
	"bytes"
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// SandboxFileService provides file APIs scoped to a sandbox.
type SandboxFileService struct {
	sandbox *Sandbox
}

// FileReadResult represents a decoded file read response.
type FileReadResult struct {
	Content *apispec.FileContentResponse
	Info    *apispec.FileInfo
	Entries []apispec.FileInfo
	Raw     *apispec.SuccessFileReadResponse
}

// FileReadOption configures file read behavior.
type FileReadOption func(*apispec.GetApiV1SandboxesIdFilesPathParams)

// WithFileStat requests file metadata.
func WithFileStat() FileReadOption {
	return func(params *apispec.GetApiV1SandboxesIdFilesPathParams) {
		stat := apispec.QueryStat(true)
		params.Stat = &stat
	}
}

// WithFileList requests directory listing.
func WithFileList() FileReadOption {
	return func(params *apispec.GetApiV1SandboxesIdFilesPathParams) {
		list := apispec.QueryList(true)
		params.List = &list
	}
}

// WithFileBinary requests base64 content encoding.
func WithFileBinary() FileReadOption {
	return func(params *apispec.GetApiV1SandboxesIdFilesPathParams) {
		binary := apispec.QueryBinary(true)
		params.Binary = &binary
	}
}

// Read reads a file or directory info.
func (s *SandboxFileService) Read(ctx context.Context, path string, opts ...FileReadOption) (*FileReadResult, error) {
	params := apispec.GetApiV1SandboxesIdFilesPathParams{}
	for _, opt := range opts {
		opt(&params)
	}
	resp, err := s.sandbox.client.api.GetApiV1SandboxesIdFilesPathWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), apispec.FilePath(path), &params)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil || resp.JSON200.Data == nil {
		return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
	}

	result := &FileReadResult{Raw: resp.JSON200}
	if content, err := resp.JSON200.Data.AsFileContentResponse(); err == nil {
		result.Content = &content
	}
	if info, err := resp.JSON200.Data.AsFileInfo(); err == nil {
		result.Info = &info
	}
	if entries, err := resp.JSON200.Data.AsSuccessFileReadResponseData1(); err == nil && entries.Entries != nil {
		result.Entries = *entries.Entries
	}
	return result, nil
}

// Stat retrieves file metadata.
func (s *SandboxFileService) Stat(ctx context.Context, path string) (*apispec.FileInfo, error) {
	result, err := s.Read(ctx, path, WithFileStat())
	if err != nil {
		return nil, err
	}
	if result.Info == nil {
		return nil, &APIError{Code: "unexpected_response", Message: "missing file info"}
	}
	return result.Info, nil
}

// List returns directory entries.
func (s *SandboxFileService) List(ctx context.Context, path string) ([]apispec.FileInfo, error) {
	result, err := s.Read(ctx, path, WithFileList())
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}

// Write writes a file.
func (s *SandboxFileService) Write(ctx context.Context, path string, data []byte) (*apispec.SuccessWrittenResponse, error) {
	body := bytes.NewReader(data)
	resp, err := s.sandbox.client.api.PostApiV1SandboxesIdFilesPathWithBodyWithResponse(
		ctx,
		apispec.SandboxID(s.sandbox.ID),
		apispec.FilePath(path),
		nil,
		"application/octet-stream",
		body,
	)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	if resp.JSON201 != nil {
		return nil, &APIError{Code: "unexpected_response", Message: "directory created instead of file"}
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Mkdir creates a directory.
func (s *SandboxFileService) Mkdir(ctx context.Context, path string, recursive bool) (*apispec.SuccessCreatedResponse, error) {
	params := apispec.PostApiV1SandboxesIdFilesPathParams{}
	mkdir := apispec.QueryMkdir(true)
	params.Mkdir = &mkdir
	if recursive {
		rec := apispec.QueryRecursive(true)
		params.Recursive = &rec
	}
	resp, err := s.sandbox.client.api.PostApiV1SandboxesIdFilesPathWithBodyWithResponse(
		ctx,
		apispec.SandboxID(s.sandbox.ID),
		apispec.FilePath(path),
		&params,
		"application/octet-stream",
		bytes.NewReader(nil),
	)
	if err != nil {
		return nil, err
	}
	if resp.JSON201 != nil {
		return resp.JSON201, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Delete deletes a file or directory.
func (s *SandboxFileService) Delete(ctx context.Context, path string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := s.sandbox.client.api.DeleteApiV1SandboxesIdFilesPathWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), apispec.FilePath(path))
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}

// Move moves a file or directory.
func (s *SandboxFileService) Move(ctx context.Context, source, destination string) (*apispec.SuccessMovedResponse, error) {
	resp, err := s.sandbox.client.api.PostApiV1SandboxesIdFilesMoveWithResponse(ctx, apispec.SandboxID(s.sandbox.ID), apispec.MoveFileRequest{
		Source:      source,
		Destination: destination,
	})
	if err != nil {
		return nil, err
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, unexpectedResponseError(resp.HTTPResponse, resp.Body)
}
