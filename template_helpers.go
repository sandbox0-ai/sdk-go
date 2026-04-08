package sandbox0

import "github.com/sandbox0-ai/sdk-go/pkg/apispec"

// TemplateOption configures a sandbox template spec.
type TemplateOption func(*apispec.SandboxTemplateSpec)

// TemplateContainerOption configures the main container spec.
type TemplateContainerOption func(*apispec.ContainerSpec)

// TemplateSidecarOption configures a sidecar container spec.
type TemplateSidecarOption func(*apispec.SidecarContainerSpec)

// TemplateSharedVolumeOption configures a shared volume spec.
type TemplateSharedVolumeOption func(*apispec.SharedVolumeSpec)

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

// TemplateResources builds a resource quota.
func TemplateResources(cpu, memory string) apispec.ResourceQuota {
	resources := apispec.ResourceQuota{}
	if cpu != "" {
		resources.CPU = apispec.NewOptString(cpu)
	}
	if memory != "" {
		resources.Memory = apispec.NewOptString(memory)
	}
	return resources
}

// TemplateMount builds a shared volume mount reference.
func TemplateMount(name, mountPath string) apispec.ContainerMountSpec {
	return apispec.ContainerMountSpec{
		Name:      name,
		MountPath: mountPath,
	}
}

// TemplateMainContainer builds a main container spec.
func TemplateMainContainer(image, cpu, memory string, opts ...TemplateContainerOption) apispec.ContainerSpec {
	container := apispec.ContainerSpec{
		Image:     image,
		Resources: TemplateResources(cpu, memory),
	}
	for _, opt := range opts {
		opt(&container)
	}
	return container
}

// TemplateSidecar builds a sidecar spec.
func TemplateSidecar(name, image, cpu, memory string, opts ...TemplateSidecarOption) apispec.SidecarContainerSpec {
	sidecar := apispec.SidecarContainerSpec{
		Name:      name,
		Image:     image,
		Resources: TemplateResources(cpu, memory),
	}
	for _, opt := range opts {
		opt(&sidecar)
	}
	return sidecar
}

// TemplateSharedVolume builds a shared volume spec.
func TemplateSharedVolume(name, mountPath string, opts ...TemplateSharedVolumeOption) apispec.SharedVolumeSpec {
	volume := apispec.SharedVolumeSpec{
		Name:      name,
		MountPath: mountPath,
	}
	for _, opt := range opts {
		opt(&volume)
	}
	return volume
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

// WithTemplateLifecycle sets the lifecycle policy.
func WithTemplateLifecycle(lifecycle apispec.LifecyclePolicy) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.Lifecycle = apispec.NewOptLifecyclePolicy(lifecycle)
	}
}

// WithTemplatePublic sets the template visibility.
func WithTemplatePublic(public bool) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.Public = apispec.NewOptBool(public)
	}
}

// WithTemplateAllowedTeams sets the allow-list for a template.
func WithTemplateAllowedTeams(teamIDs ...string) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.AllowedTeams = append([]string(nil), teamIDs...)
	}
}

// WithTemplateClusterID pins the template to a cluster.
func WithTemplateClusterID(clusterID string) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.ClusterId = apispec.NewOptString(clusterID)
	}
}

// WithTemplateSidecar appends one sidecar.
func WithTemplateSidecar(sidecar apispec.SidecarContainerSpec) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.Sidecars = append(spec.Sidecars, sidecar)
	}
}

// WithTemplateSidecars appends multiple sidecars.
func WithTemplateSidecars(sidecars ...apispec.SidecarContainerSpec) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.Sidecars = append(spec.Sidecars, sidecars...)
	}
}

// WithTemplateSharedVolume appends one shared volume.
func WithTemplateSharedVolume(volume apispec.SharedVolumeSpec) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.SharedVolumes = append(spec.SharedVolumes, volume)
	}
}

// WithTemplateSharedVolumes appends multiple shared volumes.
func WithTemplateSharedVolumes(volumes ...apispec.SharedVolumeSpec) TemplateOption {
	return func(spec *apispec.SandboxTemplateSpec) {
		spec.SharedVolumes = append(spec.SharedVolumes, volumes...)
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

// WithTemplateSidecarCommand sets the sidecar command.
func WithTemplateSidecarCommand(command ...string) TemplateSidecarOption {
	return func(sidecar *apispec.SidecarContainerSpec) {
		sidecar.Command = append([]string(nil), command...)
	}
}

// WithTemplateSidecarArgs sets the sidecar args.
func WithTemplateSidecarArgs(args ...string) TemplateSidecarOption {
	return func(sidecar *apispec.SidecarContainerSpec) {
		sidecar.Args = append([]string(nil), args...)
	}
}

// WithTemplateSidecarEnv sets the sidecar environment variables.
func WithTemplateSidecarEnv(env ...apispec.EnvVar) TemplateSidecarOption {
	return func(sidecar *apispec.SidecarContainerSpec) {
		sidecar.Env = append([]apispec.EnvVar(nil), env...)
	}
}

// WithTemplateSidecarMount appends one shared volume mount.
func WithTemplateSidecarMount(mount apispec.ContainerMountSpec) TemplateSidecarOption {
	return func(sidecar *apispec.SidecarContainerSpec) {
		sidecar.Mounts = append(sidecar.Mounts, mount)
	}
}

// WithTemplateSidecarMounts appends multiple shared volume mounts.
func WithTemplateSidecarMounts(mounts ...apispec.ContainerMountSpec) TemplateSidecarOption {
	return func(sidecar *apispec.SidecarContainerSpec) {
		sidecar.Mounts = append(sidecar.Mounts, mounts...)
	}
}

// WithTemplateSidecarReadinessProbe sets the readiness probe.
func WithTemplateSidecarReadinessProbe(probe apispec.Probe) TemplateSidecarOption {
	return func(sidecar *apispec.SidecarContainerSpec) {
		sidecar.ReadinessProbe = apispec.NewOptProbe(probe)
	}
}

// WithTemplateSidecarStartupProbe sets the startup probe.
func WithTemplateSidecarStartupProbe(probe apispec.Probe) TemplateSidecarOption {
	return func(sidecar *apispec.SidecarContainerSpec) {
		sidecar.StartupProbe = apispec.NewOptProbe(probe)
	}
}

// WithTemplateSharedVolumeCacheSize sets the shared volume cache size.
func WithTemplateSharedVolumeCacheSize(cacheSize string) TemplateSharedVolumeOption {
	return func(volume *apispec.SharedVolumeSpec) {
		volume.CacheSize = apispec.NewOptString(cacheSize)
	}
}

// WithTemplateSharedVolumeID pins the shared volume to a fixed SandboxVolume.
func WithTemplateSharedVolumeID(sandboxVolumeID string) TemplateSharedVolumeOption {
	return func(volume *apispec.SharedVolumeSpec) {
		volume.SandboxVolumeId = apispec.NewOptString(sandboxVolumeID)
	}
}

// WithTemplateSharedVolumePrefetch sets the shared volume prefetch size.
func WithTemplateSharedVolumePrefetch(prefetch int32) TemplateSharedVolumeOption {
	return func(volume *apispec.SharedVolumeSpec) {
		volume.Prefetch = apispec.NewOptInt32(prefetch)
	}
}

// WithTemplateSharedVolumeBufferSize sets the shared volume buffer size.
func WithTemplateSharedVolumeBufferSize(bufferSize string) TemplateSharedVolumeOption {
	return func(volume *apispec.SharedVolumeSpec) {
		volume.BufferSize = apispec.NewOptString(bufferSize)
	}
}

// WithTemplateSharedVolumeWriteback sets shared volume writeback mode.
func WithTemplateSharedVolumeWriteback(writeback bool) TemplateSharedVolumeOption {
	return func(volume *apispec.SharedVolumeSpec) {
		volume.Writeback = apispec.NewOptBool(writeback)
	}
}
