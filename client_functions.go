package sandbox0

import (
	"context"
	"fmt"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// FunctionSnapshotMount maps an immutable snapshot into a function runtime.
type FunctionSnapshotMount struct {
	SnapshotID string
	MountPath  string
}

// FunctionServiceSpec is the compact SDK shape for a function service.
type FunctionServiceSpec struct {
	ID              string
	DisplayName     string
	Port            int32
	Command         []string
	CWD             string
	EnvVars         map[string]string
	WarmProcessName string
	HealthPath      string
	Routes          []apispec.SandboxAppServiceRoute
}

// FunctionDeploySpec describes a snapshot-backed function deploy.
type FunctionDeploySpec struct {
	Name     string
	Slug     string
	Template string
	Service  FunctionServiceSpec
	Mounts   []FunctionSnapshotMount
	EnvVars  map[string]string
	Scale    *apispec.FunctionScalePolicy
	Activate *bool
}

type functionDeployOptions struct {
	name     string
	slug     string
	scale    *apispec.FunctionScalePolicy
	activate *bool
}

// FunctionDeployOption configures sandbox-service function deploys.
type FunctionDeployOption func(*functionDeployOptions)

// WithFunctionName sets the function display name.
func WithFunctionName(name string) FunctionDeployOption {
	return func(opts *functionDeployOptions) {
		opts.name = name
	}
}

// WithFunctionSlug sets the stable function slug.
func WithFunctionSlug(slug string) FunctionDeployOption {
	return func(opts *functionDeployOptions) {
		opts.slug = slug
	}
}

// WithFunctionScale sets the function scale-to-zero policy.
func WithFunctionScale(scale apispec.FunctionScalePolicy) FunctionDeployOption {
	return func(opts *functionDeployOptions) {
		opts.scale = &scale
	}
}

// WithFunctionActivate controls whether the deployed revision becomes active.
func WithFunctionActivate(activate bool) FunctionDeployOption {
	return func(opts *functionDeployOptions) {
		opts.activate = &activate
	}
}

// DeployFunction deploys a snapshot-backed function from a compact SDK spec.
func (c *Client) DeployFunction(ctx context.Context, spec FunctionDeploySpec) (*apispec.FunctionDeployResult, error) {
	request, err := functionDeployRequestFromSpec(spec)
	if err != nil {
		return nil, err
	}
	return c.DeployFunctionRequest(ctx, request)
}

// DeployFunctionRequest deploys a function using the generated OpenAPI request type.
func (c *Client) DeployFunctionRequest(ctx context.Context, request apispec.FunctionDeployRequest) (*apispec.FunctionDeployResult, error) {
	resp, err := c.api.APIV1FunctionsDeployPost(ctx, &request)
	if err != nil {
		return nil, err
	}
	return functionDeployResultFromResponse(resp)
}

// DeployFunctionRevision deploys a snapshot-backed revision for an existing function.
func (c *Client) DeployFunctionRevision(ctx context.Context, functionID string, spec FunctionDeploySpec) (*apispec.FunctionDeployResult, error) {
	request, err := functionDeployRequestFromSpec(spec)
	if err != nil {
		return nil, err
	}
	return c.DeployFunctionRevisionRequest(ctx, functionID, request)
}

// DeployFunctionRevisionRequest deploys a revision using the generated OpenAPI request type.
func (c *Client) DeployFunctionRevisionRequest(ctx context.Context, functionID string, request apispec.FunctionDeployRequest) (*apispec.FunctionDeployResult, error) {
	resp, err := c.api.APIV1FunctionsIDDeployPost(ctx, &request, apispec.APIV1FunctionsIDDeployPostParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	return functionDeployResultFromResponse(resp)
}

// DeployFunctionFromSandboxService creates a function revision from an existing sandbox service.
func (c *Client) DeployFunctionFromSandboxService(ctx context.Context, sandboxID, serviceID string, opts ...FunctionDeployOption) (*apispec.FunctionDeployResult, error) {
	options := functionDeployOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	request := apispec.FunctionDeployRequest{
		Source: apispec.NewOptFunctionSource(apispec.FunctionSource{
			Type: apispec.FunctionSourceTypeSandboxService,
			SandboxService: apispec.NewOptSandboxServiceFunctionSource(apispec.SandboxServiceFunctionSource{
				SandboxID: sandboxID,
				ServiceID: serviceID,
			}),
		}),
	}
	applyFunctionDeployOptions(&request, options)
	return c.DeployFunctionRequest(ctx, request)
}

// ListFunctions lists production functions for the current team.
func (c *Client) ListFunctions(ctx context.Context) ([]apispec.Function, error) {
	resp, err := c.api.APIV1FunctionsGet(ctx)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessFunctionListResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return data.Functions, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// GetFunction retrieves a function by ID or slug.
func (c *Client) GetFunction(ctx context.Context, functionID string) (*apispec.Function, error) {
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
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// UpdateFunction updates mutable function metadata.
func (c *Client) UpdateFunction(ctx context.Context, functionID string, request apispec.FunctionUpdateRequest) (*apispec.Function, error) {
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
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteFunction deletes a function identity and disables its production URL.
func (c *Client) DeleteFunction(ctx context.Context, functionID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := c.api.APIV1FunctionsIDDelete(ctx, apispec.APIV1FunctionsIDDeleteParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessDeletedResponse:
		return response, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ListFunctionRevisions lists immutable revisions for a function.
func (c *Client) ListFunctionRevisions(ctx context.Context, functionID string) ([]apispec.FunctionRevision, error) {
	resp, err := c.api.APIV1FunctionsIDRevisionsGet(ctx, apispec.APIV1FunctionsIDRevisionsGetParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessFunctionRevisionListResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return data.Revisions, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ActivateFunctionRevision makes a revision serve production function traffic.
func (c *Client) ActivateFunctionRevision(ctx context.Context, functionID, revisionID string) (*apispec.FunctionDeployResult, error) {
	request := apispec.ActivateFunctionRevisionRequest{RevisionID: revisionID}
	resp, err := c.api.APIV1FunctionsIDActiveRevisionPut(ctx, &request, apispec.APIV1FunctionsIDActiveRevisionPutParams{ID: functionID})
	if err != nil {
		return nil, err
	}
	return functionDeployResultFromResponse(resp)
}

func functionDeployRequestFromSpec(spec FunctionDeploySpec) (apispec.FunctionDeployRequest, error) {
	service, err := functionServiceFromSpec(spec.Service)
	if err != nil {
		return apispec.FunctionDeployRequest{}, err
	}
	mounts := make([]apispec.FunctionRevisionMount, 0, len(spec.Mounts))
	for _, mount := range spec.Mounts {
		if mount.SnapshotID == "" || mount.MountPath == "" {
			return apispec.FunctionDeployRequest{}, fmt.Errorf("function mount requires snapshot ID and mount path")
		}
		mounts = append(mounts, apispec.FunctionRevisionMount{
			SnapshotID: mount.SnapshotID,
			MountPath:  mount.MountPath,
			ReadOnly:   apispec.NewOptBool(true),
		})
	}
	revisionSpec := apispec.FunctionRevisionSpec{
		Template: spec.Template,
		Service:  service,
		Mounts:   mounts,
	}
	if spec.Template == "" {
		return apispec.FunctionDeployRequest{}, fmt.Errorf("function template is required")
	}
	if spec.EnvVars != nil {
		revisionSpec.EnvVars = apispec.NewOptFunctionRevisionSpecEnvVars(apispec.FunctionRevisionSpecEnvVars(spec.EnvVars))
	}
	request := apispec.FunctionDeployRequest{
		Source: apispec.NewOptFunctionSource(apispec.FunctionSource{
			Type: apispec.FunctionSourceTypeSnapshot,
		}),
		Spec: apispec.NewOptFunctionRevisionSpec(revisionSpec),
	}
	applyFunctionDeployOptions(&request, functionDeployOptions{
		name:     spec.Name,
		slug:     spec.Slug,
		scale:    spec.Scale,
		activate: spec.Activate,
	})
	return request, nil
}

func functionServiceFromSpec(spec FunctionServiceSpec) (apispec.SandboxAppService, error) {
	if spec.ID == "" {
		spec.ID = "app"
	}
	if spec.Port <= 0 {
		return apispec.SandboxAppService{}, fmt.Errorf("function service port is required")
	}
	runtime, err := functionRuntimeFromSpec(spec)
	if err != nil {
		return apispec.SandboxAppService{}, err
	}
	service := apispec.SandboxAppService{
		ID:      spec.ID,
		Port:    spec.Port,
		Runtime: apispec.NewOptSandboxAppServiceRuntime(runtime),
		Ingress: apispec.SandboxAppServiceIngress{
			Public: true,
			Routes: append([]apispec.SandboxAppServiceRoute(nil), spec.Routes...),
		},
	}
	if spec.DisplayName != "" {
		service.DisplayName = apispec.NewOptString(spec.DisplayName)
	}
	if spec.HealthPath != "" {
		service.HealthCheck = apispec.NewOptSandboxAppServiceHealth(apispec.SandboxAppServiceHealth{
			Path: apispec.NewOptString(spec.HealthPath),
		})
	}
	return service, nil
}

func functionRuntimeFromSpec(spec FunctionServiceSpec) (apispec.SandboxAppServiceRuntime, error) {
	switch {
	case len(spec.Command) > 0:
		runtime := apispec.SandboxAppServiceRuntime{
			Type:    apispec.SandboxAppServiceRuntimeTypeCmd,
			Command: append([]string(nil), spec.Command...),
		}
		if spec.CWD != "" {
			runtime.Cwd = apispec.NewOptString(spec.CWD)
		}
		if spec.EnvVars != nil {
			runtime.EnvVars = apispec.NewOptSandboxAppServiceRuntimeEnvVars(apispec.SandboxAppServiceRuntimeEnvVars(spec.EnvVars))
		}
		return runtime, nil
	case spec.WarmProcessName != "":
		return apispec.SandboxAppServiceRuntime{
			Type:            apispec.SandboxAppServiceRuntimeTypeWarmProcess,
			WarmProcessName: apispec.NewOptString(spec.WarmProcessName),
		}, nil
	default:
		return apispec.SandboxAppServiceRuntime{}, fmt.Errorf("function service command or warm process name is required")
	}
}

func applyFunctionDeployOptions(request *apispec.FunctionDeployRequest, opts functionDeployOptions) {
	if opts.name != "" {
		request.Name = apispec.NewOptString(opts.name)
	}
	if opts.slug != "" {
		request.Slug = apispec.NewOptString(opts.slug)
	}
	if opts.scale != nil {
		request.Scale = apispec.NewOptFunctionScalePolicy(*opts.scale)
	}
	if opts.activate != nil {
		request.Activate = apispec.NewOptBool(*opts.activate)
	}
}

func functionDeployResultFromResponse(resp any) (*apispec.FunctionDeployResult, error) {
	switch response := resp.(type) {
	case *apispec.SuccessFunctionDeployResultResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}
