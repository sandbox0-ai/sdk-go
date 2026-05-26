package sandbox0

import (
	"context"
	"fmt"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// RunSnapshotMount maps an immutable snapshot into a run runtime.
type RunSnapshotMount struct {
	SnapshotID string
	MountPath  string
}

// RunServiceSpec is the compact SDK shape for a run service.
type RunServiceSpec struct {
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

// RunDeploySpec describes a snapshot-backed run deploy.
type RunDeploySpec struct {
	Name     string
	Slug     string
	Template string
	Service  RunServiceSpec
	Mounts   []RunSnapshotMount
	EnvVars  map[string]string
	Scale    *apispec.RunScalePolicy
	Activate *bool
}

type runDeployOptions struct {
	name     string
	slug     string
	scale    *apispec.RunScalePolicy
	activate *bool
}

// RunDeployOption configures sandbox-service run deploys.
type RunDeployOption func(*runDeployOptions)

// WithRunName sets the run display name.
func WithRunName(name string) RunDeployOption {
	return func(opts *runDeployOptions) {
		opts.name = name
	}
}

// WithRunSlug sets the stable run slug.
func WithRunSlug(slug string) RunDeployOption {
	return func(opts *runDeployOptions) {
		opts.slug = slug
	}
}

// WithRunScale sets the run scale-to-zero policy.
func WithRunScale(scale apispec.RunScalePolicy) RunDeployOption {
	return func(opts *runDeployOptions) {
		opts.scale = &scale
	}
}

// WithRunActivate controls whether the deployed revision becomes active.
func WithRunActivate(activate bool) RunDeployOption {
	return func(opts *runDeployOptions) {
		opts.activate = &activate
	}
}

// DeployRun deploys a snapshot-backed run from a compact SDK spec.
func (c *Client) DeployRun(ctx context.Context, spec RunDeploySpec) (*apispec.RunDeployResult, error) {
	request, err := runDeployRequestFromSpec(spec)
	if err != nil {
		return nil, err
	}
	return c.DeployRunRequest(ctx, request)
}

// DeployRunRequest deploys a run using the generated OpenAPI request type.
func (c *Client) DeployRunRequest(ctx context.Context, request apispec.RunDeployRequest) (*apispec.RunDeployResult, error) {
	resp, err := c.api.APIV1RunsDeployPost(ctx, &request)
	if err != nil {
		return nil, err
	}
	return runDeployResultFromResponse(resp)
}

// DeployRunRevision deploys a snapshot-backed revision for an existing run.
func (c *Client) DeployRunRevision(ctx context.Context, runID string, spec RunDeploySpec) (*apispec.RunDeployResult, error) {
	request, err := runDeployRequestFromSpec(spec)
	if err != nil {
		return nil, err
	}
	return c.DeployRunRevisionRequest(ctx, runID, request)
}

// DeployRunRevisionRequest deploys a revision using the generated OpenAPI request type.
func (c *Client) DeployRunRevisionRequest(ctx context.Context, runID string, request apispec.RunDeployRequest) (*apispec.RunDeployResult, error) {
	resp, err := c.api.APIV1RunsIDDeployPost(ctx, &request, apispec.APIV1RunsIDDeployPostParams{ID: runID})
	if err != nil {
		return nil, err
	}
	return runDeployResultFromResponse(resp)
}

// DeployRunFromSandboxService creates a run revision from an existing sandbox service.
func (c *Client) DeployRunFromSandboxService(ctx context.Context, sandboxID, serviceID string, opts ...RunDeployOption) (*apispec.RunDeployResult, error) {
	options := runDeployOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	request := apispec.RunDeployRequest{
		Source: apispec.NewOptRunSource(apispec.RunSource{
			Type: apispec.RunSourceTypeSandboxService,
			SandboxService: apispec.NewOptSandboxServiceRunSource(apispec.SandboxServiceRunSource{
				SandboxID: sandboxID,
				ServiceID: serviceID,
			}),
		}),
	}
	applyRunDeployOptions(&request, options)
	return c.DeployRunRequest(ctx, request)
}

// ListRuns lists production runs for the current team.
func (c *Client) ListRuns(ctx context.Context) ([]apispec.Run, error) {
	resp, err := c.api.APIV1RunsGet(ctx)
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessRunListResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return data.Runs, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// GetRun retrieves a run by ID or slug.
func (c *Client) GetRun(ctx context.Context, runID string) (*apispec.Run, error) {
	resp, err := c.api.APIV1RunsIDGet(ctx, apispec.APIV1RunsIDGetParams{ID: runID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessRunResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// UpdateRun updates mutable run metadata.
func (c *Client) UpdateRun(ctx context.Context, runID string, request apispec.RunUpdateRequest) (*apispec.Run, error) {
	resp, err := c.api.APIV1RunsIDPut(ctx, &request, apispec.APIV1RunsIDPutParams{ID: runID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessRunResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// DeleteRun deletes a run identity and disables its production URL.
func (c *Client) DeleteRun(ctx context.Context, runID string) (*apispec.SuccessDeletedResponse, error) {
	resp, err := c.api.APIV1RunsIDDelete(ctx, apispec.APIV1RunsIDDeleteParams{ID: runID})
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

// ListRunRevisions lists immutable revisions for a run.
func (c *Client) ListRunRevisions(ctx context.Context, runID string) ([]apispec.RunRevision, error) {
	resp, err := c.api.APIV1RunsIDRevisionsGet(ctx, apispec.APIV1RunsIDRevisionsGetParams{ID: runID})
	if err != nil {
		return nil, err
	}
	switch response := resp.(type) {
	case *apispec.SuccessRunRevisionListResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return data.Revisions, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}

// ActivateRunRevision makes a revision serve production run traffic.
func (c *Client) ActivateRunRevision(ctx context.Context, runID, revisionID string) (*apispec.RunDeployResult, error) {
	request := apispec.ActivateRunRevisionRequest{RevisionID: revisionID}
	resp, err := c.api.APIV1RunsIDActiveRevisionPut(ctx, &request, apispec.APIV1RunsIDActiveRevisionPutParams{ID: runID})
	if err != nil {
		return nil, err
	}
	return runDeployResultFromResponse(resp)
}

func runDeployRequestFromSpec(spec RunDeploySpec) (apispec.RunDeployRequest, error) {
	service, err := runServiceFromSpec(spec.Service)
	if err != nil {
		return apispec.RunDeployRequest{}, err
	}
	mounts := make([]apispec.RunRevisionMount, 0, len(spec.Mounts))
	for _, mount := range spec.Mounts {
		if mount.SnapshotID == "" || mount.MountPath == "" {
			return apispec.RunDeployRequest{}, fmt.Errorf("run mount requires snapshot ID and mount path")
		}
		mounts = append(mounts, apispec.RunRevisionMount{
			SnapshotID: mount.SnapshotID,
			MountPath:  mount.MountPath,
			ReadOnly:   apispec.NewOptBool(true),
		})
	}
	revisionSpec := apispec.RunRevisionSpec{
		Template: spec.Template,
		Service:  service,
		Mounts:   mounts,
	}
	if spec.Template == "" {
		return apispec.RunDeployRequest{}, fmt.Errorf("run template is required")
	}
	if spec.EnvVars != nil {
		revisionSpec.EnvVars = apispec.NewOptRunRevisionSpecEnvVars(apispec.RunRevisionSpecEnvVars(spec.EnvVars))
	}
	request := apispec.RunDeployRequest{
		Source: apispec.NewOptRunSource(apispec.RunSource{
			Type: apispec.RunSourceTypeSnapshot,
		}),
		Spec: apispec.NewOptRunRevisionSpec(revisionSpec),
	}
	applyRunDeployOptions(&request, runDeployOptions{
		name:     spec.Name,
		slug:     spec.Slug,
		scale:    spec.Scale,
		activate: spec.Activate,
	})
	return request, nil
}

func runServiceFromSpec(spec RunServiceSpec) (apispec.SandboxAppService, error) {
	if spec.ID == "" {
		spec.ID = "app"
	}
	if spec.Port <= 0 {
		return apispec.SandboxAppService{}, fmt.Errorf("run service port is required")
	}
	runtime, err := runRuntimeFromSpec(spec)
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

func runRuntimeFromSpec(spec RunServiceSpec) (apispec.SandboxAppServiceRuntime, error) {
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
		return apispec.SandboxAppServiceRuntime{}, fmt.Errorf("run service command or warm process name is required")
	}
}

func applyRunDeployOptions(request *apispec.RunDeployRequest, opts runDeployOptions) {
	if opts.name != "" {
		request.Name = apispec.NewOptString(opts.name)
	}
	if opts.slug != "" {
		request.Slug = apispec.NewOptString(opts.slug)
	}
	if opts.scale != nil {
		request.Scale = apispec.NewOptRunScalePolicy(*opts.scale)
	}
	if opts.activate != nil {
		request.Activate = apispec.NewOptBool(*opts.activate)
	}
}

func runDeployResultFromResponse(resp any) (*apispec.RunDeployResult, error) {
	switch response := resp.(type) {
	case *apispec.SuccessRunDeployResultResponse:
		data, ok := response.Data.Get()
		if !ok {
			return nil, unexpectedResponseError(response)
		}
		return &data, nil
	default:
		return nil, apiErrorFromResponse(response)
	}
}
