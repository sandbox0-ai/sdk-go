package sandbox0

import (
	"context"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// FunctionCreateResult contains the function, first revision, and production alias.
type FunctionCreateResult struct {
	Function apispec.FunctionRecord
	Revision apispec.FunctionRevision
	Alias    apispec.FunctionAlias
}

// FunctionRevisionCreateResult contains a newly created function revision.
type FunctionRevisionCreateResult struct {
	Revision apispec.FunctionRevision
	Promoted bool
}

// FunctionSource builds a function source reference from a sandbox service.
func FunctionSource(sandboxID, serviceID string) apispec.FunctionSourceRequest {
	return apispec.FunctionSourceRequest{
		SandboxID: sandboxID,
		ServiceID: serviceID,
	}
}

// FunctionCreateOption configures CreateFunctionFromSandbox.
type FunctionCreateOption func(*apispec.FunctionCreateRequest)

// WithFunctionName sets the function display name.
func WithFunctionName(name string) FunctionCreateOption {
	return func(req *apispec.FunctionCreateRequest) {
		req.Name = apispec.NewOptString(name)
	}
}

// FunctionUpdateOption configures UpdateFunction.
type FunctionUpdateOption func(*apispec.FunctionUpdateRequest)

// WithFunctionUpdateName sets the function display name without changing slug or host.
func WithFunctionUpdateName(name string) FunctionUpdateOption {
	return func(req *apispec.FunctionUpdateRequest) {
		req.Name = apispec.NewOptString(name)
	}
}

// WithFunctionEnabled controls whether a function serves host traffic.
func WithFunctionEnabled(enabled bool) FunctionUpdateOption {
	return func(req *apispec.FunctionUpdateRequest) {
		req.Enabled = apispec.NewOptBool(enabled)
	}
}

// FunctionRevisionCreateOption configures CreateFunctionRevisionFromSandbox.
type FunctionRevisionCreateOption func(*apispec.FunctionRevisionCreateRequest)

// WithFunctionRevisionPromote controls whether the production alias moves to the new revision.
func WithFunctionRevisionPromote(promote bool) FunctionRevisionCreateOption {
	return func(req *apispec.FunctionRevisionCreateRequest) {
		req.Promote = apispec.NewOptBool(promote)
	}
}

// ListFunctions lists functions for the current team.
func (c *Client) ListFunctions(ctx context.Context) ([]apispec.FunctionRecord, error) {
	resp, err := c.api.APIV1FunctionsGet(ctx)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return data.Functions, nil
}

// GetFunction retrieves a function by ID or slug.
func (c *Client) GetFunction(ctx context.Context, functionID string) (*apispec.FunctionRecord, error) {
	resp, err := c.api.APIV1FunctionsIDGet(ctx, apispec.APIV1FunctionsIDGetParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessFunctionResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data.Function, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// UpdateFunction updates mutable function metadata and serving state.
func (c *Client) UpdateFunction(ctx context.Context, functionID string, request apispec.FunctionUpdateRequest) (*apispec.FunctionRecord, error) {
	resp, err := c.api.APIV1FunctionsIDPut(ctx, &request, apispec.APIV1FunctionsIDPutParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessFunctionResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data.Function, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// UpdateFunctionWithOptions updates mutable function fields using option helpers.
func (c *Client) UpdateFunctionWithOptions(ctx context.Context, functionID string, opts ...FunctionUpdateOption) (*apispec.FunctionRecord, error) {
	var req apispec.FunctionUpdateRequest
	for _, opt := range opts {
		opt(&req)
	}
	return c.UpdateFunction(ctx, functionID, req)
}

// DeleteFunction soft-deletes a function and removes it from normal list/get/host traffic.
func (c *Client) DeleteFunction(ctx context.Context, functionID string) (*apispec.FunctionRecord, error) {
	resp, err := c.api.APIV1FunctionsIDDelete(ctx, apispec.APIV1FunctionsIDDeleteParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessFunctionResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data.Function, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// CreateFunction creates a function from a sandbox service.
func (c *Client) CreateFunction(ctx context.Context, request apispec.FunctionCreateRequest) (*FunctionCreateResult, error) {
	resp, err := c.api.APIV1FunctionsPost(ctx, &request)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessFunctionCreateResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &FunctionCreateResult{
			Function: data.Function,
			Revision: data.Revision,
			Alias:    data.Alias,
		}, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// CreateFunctionFromSandbox creates a function from a publishable sandbox service.
func (c *Client) CreateFunctionFromSandbox(ctx context.Context, sandboxID, serviceID string, opts ...FunctionCreateOption) (*FunctionCreateResult, error) {
	req := apispec.FunctionCreateRequest{
		Source: FunctionSource(sandboxID, serviceID),
	}
	for _, opt := range opts {
		opt(&req)
	}
	return c.CreateFunction(ctx, req)
}

// ListFunctionRevisions lists revisions for a function.
func (c *Client) ListFunctionRevisions(ctx context.Context, functionID string) ([]apispec.FunctionRevision, error) {
	resp, err := c.api.APIV1FunctionsIDRevisionsGet(ctx, apispec.APIV1FunctionsIDRevisionsGetParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return data.Revisions, nil
}

// GetFunctionRevision retrieves a function revision by revision number.
func (c *Client) GetFunctionRevision(ctx context.Context, functionID string, revisionNumber int32) (*apispec.FunctionRevision, error) {
	resp, err := c.api.APIV1FunctionsIDRevisionsRevisionNumberGet(ctx, apispec.APIV1FunctionsIDRevisionsRevisionNumberGetParams{
		ID:             functionID,
		RevisionNumber: revisionNumber,
	})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessFunctionRevisionResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data.Revision, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// CreateFunctionRevision creates a new function revision from a sandbox service.
func (c *Client) CreateFunctionRevision(ctx context.Context, functionID string, request apispec.FunctionRevisionCreateRequest) (*FunctionRevisionCreateResult, error) {
	resp, err := c.api.APIV1FunctionsIDRevisionsPost(ctx, &request, apispec.APIV1FunctionsIDRevisionsPostParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &FunctionRevisionCreateResult{
		Revision: data.Revision,
		Promoted: data.Promoted,
	}, nil
}

// CreateFunctionRevisionFromSandbox creates a revision from a publishable sandbox service.
func (c *Client) CreateFunctionRevisionFromSandbox(ctx context.Context, functionID, sandboxID, serviceID string, opts ...FunctionRevisionCreateOption) (*FunctionRevisionCreateResult, error) {
	req := apispec.FunctionRevisionCreateRequest{
		Source: FunctionSource(sandboxID, serviceID),
	}
	for _, opt := range opts {
		opt(&req)
	}
	return c.CreateFunctionRevision(ctx, functionID, req)
}

// ListFunctionAliases lists aliases for a function.
func (c *Client) ListFunctionAliases(ctx context.Context, functionID string) ([]apispec.FunctionAlias, error) {
	resp, err := c.api.APIV1FunctionsIDAliasesGet(ctx, apispec.APIV1FunctionsIDAliasesGetParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return data.Aliases, nil
}

// GetFunctionAlias retrieves an alias for a function.
func (c *Client) GetFunctionAlias(ctx context.Context, functionID, alias string) (*apispec.FunctionAlias, error) {
	resp, err := c.api.APIV1FunctionsIDAliasesAliasGet(ctx, apispec.APIV1FunctionsIDAliasesAliasGetParams{ID: functionID, Alias: alias})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessFunctionAliasResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data.Alias, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// SetFunctionAlias points an alias at a revision number.
func (c *Client) SetFunctionAlias(ctx context.Context, functionID, alias string, revisionNumber int32) (*apispec.FunctionAlias, error) {
	resp, err := c.api.APIV1FunctionsIDAliasesAliasPut(ctx, &apispec.FunctionAliasUpdateRequest{
		RevisionNumber: revisionNumber,
	}, apispec.APIV1FunctionsIDAliasesAliasPutParams{ID: functionID, Alias: alias})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data.Alias, nil
}

// GetFunctionRuntime returns the currently restored runtime status for a function.
func (c *Client) GetFunctionRuntime(ctx context.Context, functionID string) (*apispec.FunctionRuntimeStatus, error) {
	resp, err := c.api.APIV1FunctionsIDRuntimeGet(ctx, apispec.APIV1FunctionsIDRuntimeGetParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data.Runtime, nil
}

// RestartFunctionRuntime deletes the current runtime sandbox and leaves the function idle.
func (c *Client) RestartFunctionRuntime(ctx context.Context, functionID string) (*apispec.FunctionRuntimeStatus, error) {
	resp, err := c.api.APIV1FunctionsIDRuntimeRestartPost(ctx, apispec.APIV1FunctionsIDRuntimeRestartPostParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data.Runtime, nil
}

// RecycleFunctionRuntime is an alias for restarting the current runtime sandbox.
func (c *Client) RecycleFunctionRuntime(ctx context.Context, functionID string) (*apispec.FunctionRuntimeStatus, error) {
	resp, err := c.api.APIV1FunctionsIDRuntimeRecyclePost(ctx, apispec.APIV1FunctionsIDRuntimeRecyclePostParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, unexpectedResponseError(resp)
	}
	data, ok := resp.Data.Get()
	if !ok {
		return nil, unexpectedResponseError(resp)
	}
	return &data.Runtime, nil
}
