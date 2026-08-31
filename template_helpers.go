package sandbox0

import "github.com/sandbox0-ai/sdk-go/pkg/apispec"

// TemplateOption configures a sandbox template spec.
type TemplateOption func(*apispec.SandboxTemplateSpec)

// TemplateContainerOption configures the main container spec.
type TemplateContainerOption func(*apispec.ContainerSpec)

// TemplateEphemeralMount builds a claim-lifetime tmpfs mount spec.
func TemplateEphemeralMount(mountPath, sizeLimit string) apispec.EphemeralMountSpec {
	return apispec.EphemeralMountSpec{
		MountPath: mountPath,
		SizeLimit: sizeLimit,
	}
}

// NewTemplateSpec builds a template spec around a main container.
func NewTemplateSpec(main apispec.ContainerSpec, opts ...TemplateOption) apispec.SandboxTemplateSpec {
	spec := apispec.SandboxTemplateSpec{
		MainContainer: main,
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

// NewTemplateFromSandboxCreateRequest builds a template request from a sandbox root filesystem.
func NewTemplateFromSandboxCreateRequest(
	templateID string,
	sandboxID string,
	overrides *apispec.TemplateFromSandboxSpecOverrides,
) apispec.TemplateFromSandboxCreateRequest {
	request := apispec.TemplateFromSandboxCreateRequest{
		TemplateID: templateID,
		SandboxID:  sandboxID,
	}
	if overrides != nil {
		request.SpecOverrides = apispec.NewOptTemplateFromSandboxSpecOverrides(*overrides)
	}
	return request
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

// WithTemplateNetwork sets the network policy.
func WithTemplateNetwork(network apispec.SandboxNetworkPolicy) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.Network = apispec.NewOptSandboxNetworkPolicy(network)
	}
}

// WithTemplateEphemeralMount appends one claim-lifetime tmpfs mount.
func WithTemplateEphemeralMount(mount apispec.EphemeralMountSpec) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.EphemeralMounts = append(spec.EphemeralMounts, mount)
	}
}

// WithTemplateEphemeralMounts appends multiple claim-lifetime tmpfs mounts.
func WithTemplateEphemeralMounts(mounts ...apispec.EphemeralMountSpec) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.EphemeralMounts = append(spec.EphemeralMounts, mounts...)
	}
}

// WithTemplateContainerEnv sets main container environment variables.
func WithTemplateContainerEnv(env ...apispec.EnvVar) TemplateContainerOption {
	return func(container *apispec.ContainerSpec) {
		container.Env = append([]apispec.EnvVar(nil), env...)
	}
}

// WithTemplateContainerSecurityClass sets the immutable gVisor guest privilege class.
func WithTemplateContainerSecurityClass(securityClass apispec.ContainerSpecSecurityClass) TemplateContainerOption {
	return func(container *apispec.ContainerSpec) {
		container.SecurityClass = apispec.NewOptContainerSpecSecurityClass(securityClass)
	}
}
