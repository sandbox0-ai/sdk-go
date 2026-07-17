package sandbox0

import "github.com/sandbox0-ai/sdk-go/pkg/apispec"

// TemplateOption configures a sandbox template spec.
type TemplateOption func(*apispec.SandboxTemplateSpec)

// TemplateContainerOption configures the main container spec.
type TemplateContainerOption func(*apispec.ContainerSpec)

// TemplateEmptyDirMount builds an emptyDir mount spec.
func TemplateEmptyDirMount(mountPath, sizeLimit string) apispec.EmptyDirMountSpec {
	mount := apispec.EmptyDirMountSpec{MountPath: mountPath}
	if sizeLimit != "" {
		mount.SizeLimit = apispec.NewOptString(sizeLimit)
	}
	return mount
}

// NewTemplateSpec builds a template spec around a main container.
func NewTemplateSpec(main apispec.ContainerSpec, opts ...TemplateOption) apispec.SandboxTemplateSpec {
	spec := apispec.SandboxTemplateSpec{
		MainContainer: apispec.NewOptContainerSpec(main),
	}
	for _, opt := range opts {
		opt(&spec)
	}
	return spec
}

// NewTemplateCreateRequest builds a template create request.
func NewTemplateCreateRequest(templateID string, main apispec.ContainerSpec, opts ...TemplateOption) apispec.TemplateCreateRequest {
	return apispec.TemplateCreateRequest{
		TemplateID: templateID,
		Spec:       NewTemplateSpec(main, opts...),
	}
}

// NewTemplateUpdateRequest builds a template update request.
func NewTemplateUpdateRequest(spec apispec.SandboxTemplateSpec) apispec.TemplateUpdateRequest {
	return apispec.TemplateUpdateRequest{Spec: spec}
}

// TemplateResources builds a memory-only resource quota.
func TemplateResources(memory string) apispec.ResourceQuota {
	return apispec.ResourceQuota{Memory: memory}
}

// TemplateMainContainer builds a main container spec. Sandbox0 derives CPU from platform configuration.
func TemplateMainContainer(image, memory string, opts ...TemplateContainerOption) apispec.ContainerSpec {
	container := apispec.ContainerSpec{
		Image:     image,
		Resources: TemplateResources(memory),
	}
	for _, opt := range opts {
		opt(&container)
	}
	return container
}

// WithTemplateDescription sets the template description.
func WithTemplateDescription(description string) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.Description = apispec.NewOptString(description)
	}
}

// WithTemplateDisplayName sets the template display name.
func WithTemplateDisplayName(displayName string) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.DisplayName = apispec.NewOptString(displayName)
	}
}

// WithTemplateTags sets the template tags.
func WithTemplateTags(tags ...string) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.Tags = append([]string(nil), tags...)
	}
}

// WithTemplateEnvVars sets template-level environment variables.
func WithTemplateEnvVars(envVars map[string]string) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		if envVars == nil {
			spec.EnvVars = apispec.OptSandboxTemplateSpecEnvVars{}
			return
		}
		copied := make(apispec.SandboxTemplateSpecEnvVars, len(envVars))
		for key, value := range envVars {
			copied[key] = value
		}
		spec.EnvVars = apispec.NewOptSandboxTemplateSpecEnvVars(copied)
	}
}

// WithTemplatePool sets the pool strategy.
func WithTemplatePool(pool apispec.PoolStrategy) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.Pool = apispec.NewOptPoolStrategy(pool)
	}
}

// WithTemplateNetwork sets the network policy.
func WithTemplateNetwork(network apispec.SandboxNetworkPolicy) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.Network = apispec.NewOptSandboxNetworkPolicy(network)
	}
}

// WithTemplateClusterID pins the template to a cluster.
func WithTemplateClusterID(clusterID string) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.ClusterId = apispec.NewOptString(clusterID)
	}
}

// WithTemplatePod sets pod-level template overrides.
func WithTemplatePod(pod apispec.PodSpecOverride) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.Pod = apispec.NewOptPodSpecOverride(pod)
	}
}

// WithTemplateEmptyDirMount appends one pod emptyDir mount.
func WithTemplateEmptyDirMount(mount apispec.EmptyDirMountSpec) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		pod, ok := spec.Pod.Get()
		if !ok {
			pod = apispec.PodSpecOverride{}
		}
		pod.EmptyDirMounts = append(pod.EmptyDirMounts, mount)
		spec.Pod = apispec.NewOptPodSpecOverride(pod)
	}
}

// WithTemplateEmptyDirMounts appends multiple pod emptyDir mounts.
func WithTemplateEmptyDirMounts(mounts ...apispec.EmptyDirMountSpec) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		pod, ok := spec.Pod.Get()
		if !ok {
			pod = apispec.PodSpecOverride{}
		}
		pod.EmptyDirMounts = append(pod.EmptyDirMounts, mounts...)
		spec.Pod = apispec.NewOptPodSpecOverride(pod)
	}
}

// WithTemplateContainerEnv sets main container environment variables.
func WithTemplateContainerEnv(env ...apispec.EnvVar) TemplateContainerOption {
	return func(container *apispec.ContainerSpec) {
		container.Env = append([]apispec.EnvVar(nil), env...)
	}
}

// WithTemplateContainerImagePullPolicy sets the main container pull policy.
func WithTemplateContainerImagePullPolicy(policy string) TemplateContainerOption {
	return func(container *apispec.ContainerSpec) {
		container.ImagePullPolicy = apispec.NewOptString(policy)
	}
}

// WithTemplateContainerSecurityContext sets the main container security context.
func WithTemplateContainerSecurityContext(securityContext apispec.SecurityContext) TemplateContainerOption {
	return func(container *apispec.ContainerSpec) {
		container.SecurityContext = apispec.NewOptSecurityContext(securityContext)
	}
}
