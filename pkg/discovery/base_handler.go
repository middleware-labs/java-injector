package discovery

// CacheDetailMap maps a Process Detail key to the corresponding
// ProcessCacheEntry field accessor, used to restore per-language details
// from cache without duplicating the mapping in every handler.
type CacheDetailMap struct {
	Key      string
	Accessor func(ProcessCacheEntry) any
}

// HandlerConfig holds all language-specific constants that vary across
// handlers but follow the same structural pattern. Each handler provides
// one of these to BaseHandler at construction time.
type HandlerConfig struct {
	Language           Language
	RuntimeName        string
	RuntimeDescription string

	// LanguageString is the string used in ServiceSetting.Language (e.g. "java", "node").
	// Defaults to RuntimeName if empty.
	LanguageString string

	// DefaultServiceType is the service type when no systemd unit or container
	// is present. Most handlers use "system"; Java uses "standalone".
	DefaultServiceType string

	// ContainerKeyPrefix is the prefix for the ServiceSetting key when the
	// process is in a container. Most use "container"; Java/Python/Rust use "docker".
	ContainerKeyPrefix string

	// CacheDetailMapping defines per-language Detail keys to restore from
	// cache, beyond the three common ones (systemd_unit, explicit_service_name,
	// working_directory) which are always restored.
	CacheDetailMapping []CacheDetailMap

	// ExtraCacheWriter populates additional ProcessCacheEntry fields beyond
	// the 12 common ones. Called during WriteCacheEntry. Optional.
	ExtraCacheWriter func(proc *Process, entry *ProcessCacheEntry)

	// RuntimeNameFromCache overrides RuntimeName on the cache path. PHP uses
	// this to set RuntimeName to cached.ServiceType (php-fpm, php-cli, etc.).
	RuntimeNameFromCache func(ProcessCacheEntry) string

	// OverrideServiceNameOnContainer, when true, sets ServiceSetting.ServiceName
	// to the container name (Node, Go, Ruby do this).
	OverrideServiceNameOnContainer bool
}

// BaseHandler provides shared Enrich/ToServiceSetting/PassesFilter logic.
// Language handlers embed this and call its methods to avoid boilerplate.
type BaseHandler struct {
	Config HandlerConfig
}

func (b *BaseHandler) languageString() string {
	if b.Config.LanguageString != "" {
		return b.Config.LanguageString
	}
	return b.Config.RuntimeName
}

func (b *BaseHandler) defaultServiceType() string {
	if b.Config.DefaultServiceType != "" {
		return b.Config.DefaultServiceType
	}
	return "system"
}

func (b *BaseHandler) containerKeyPrefix() string {
	if b.Config.ContainerKeyPrefix != "" {
		return b.Config.ContainerKeyPrefix
	}
	return "container"
}

// BuildProcessFromCache constructs a Process from a cache hit. The three
// common Detail keys plus all handler-specific CacheDetailMapping entries
// are restored.
func (b *BaseHandler) BuildProcessFromCache(info *ProcessInfo, cached ProcessCacheEntry, createTime int64) *Process {
	runtimeName := b.Config.RuntimeName
	if b.Config.RuntimeNameFromCache != nil {
		runtimeName = b.Config.RuntimeNameFromCache(cached)
	}

	proc := &Process{
		PID:            info.PID,
		ParentPID:      readProcessPPID(info.PID),
		ExecutableName: info.ExeName,
		ExecutablePath: info.ExePath,
		Command:        info.CmdLine,
		CommandLine:    info.CmdLine,
		Owner:          cached.Owner,
		CreateTime:     timeFromMillis(createTime),
		Status:         readProcessStatus(info.PID),
		Language:       b.Config.Language,

		ServiceName:        cached.ServiceName,
		RuntimeName:        runtimeName,
		RuntimeVersion:     cached.RuntimeVersion,
		RuntimeDescription: b.Config.RuntimeDescription,

		HasAgent:          cached.HasAgent,
		IsMiddlewareAgent: cached.IsMiddlewareAgent,
		AgentPath:         cached.AgentPath,
		AgentType:         cached.AgentType,
		ContainerInfo:     cached.ContainerInfo,

		Details: map[string]any{
			DetailSystemdUnit:         cached.SystemdUnit,
			DetailExplicitServiceName: cached.ExplicitServiceName,
			DetailWorkingDirectory:    cached.WorkingDirectory,
		},
	}

	for _, m := range b.Config.CacheDetailMapping {
		proc.Details[m.Key] = m.Accessor(cached)
	}

	return proc
}

// BuildProcess constructs a fresh Process for the slow (uncached) path.
func (b *BaseHandler) BuildProcess(info *ProcessInfo, owner string, createTime int64, runtimeVersion string) *Process {
	return &Process{
		PID:            info.PID,
		ParentPID:      readProcessPPID(info.PID),
		ExecutableName: info.ExeName,
		ExecutablePath: info.ExePath,
		Command:        info.CmdLine,
		CommandLine:    info.CmdLine,
		CommandArgs:    info.CmdArgs,
		Owner:          owner,
		CreateTime:     timeFromMillis(createTime),
		Status:         readProcessStatus(info.PID),
		Language:       b.Config.Language,

		RuntimeName:        b.Config.RuntimeName,
		RuntimeVersion:     runtimeVersion,
		RuntimeDescription: b.Config.RuntimeDescription,

		Details: make(map[string]any),
	}
}

// ApplyContainerInfo attaches container info to the process if discovery
// options require it. Returns true if the process should be skipped
// (container excluded by filter).
func (b *BaseHandler) ApplyContainerInfo(proc *Process, opts DiscoveryOptions, detector *ContainerDetector) bool {
	if !opts.IncludeContainerInfo && !opts.ExcludeContainers {
		return false
	}
	containerInfo, err := detector.IsProcessInContainer(proc.PID)
	if err != nil {
		return false
	}
	proc.ContainerInfo = containerInfo
	return opts.ExcludeContainers && containerInfo.IsContainer
}

// WriteCacheEntry builds a ProcessCacheEntry from a fully-enriched Process,
// populating the 12 common fields plus any extras via ExtraCacheWriter.
func (b *BaseHandler) WriteCacheEntry(proc *Process) ProcessCacheEntry {
	entry := ProcessCacheEntry{
		ServiceName:         proc.ServiceName,
		ServiceType:         proc.DetailString(DetailProcessManager),
		RuntimeVersion:      proc.RuntimeVersion,
		EntryPoint:          proc.DetailString(DetailEntryPoint),
		HasAgent:            proc.HasAgent,
		IsMiddlewareAgent:   proc.IsMiddlewareAgent,
		AgentPath:           proc.AgentPath,
		AgentType:           proc.AgentType,
		ContainerInfo:       proc.ContainerInfo,
		Owner:               proc.Owner,
		SystemdUnit:         proc.DetailString(DetailSystemdUnit),
		ExplicitServiceName: proc.DetailString(DetailExplicitServiceName),
		WorkingDirectory:    proc.DetailString(DetailWorkingDirectory),
	}
	if b.Config.ExtraCacheWriter != nil {
		b.Config.ExtraCacheWriter(proc, &entry)
	}
	return entry
}

// DefaultPassesFilter implements the common owner-based filtering logic
// shared by most handlers (all except Java which has extra agent filters).
func (b *BaseHandler) DefaultPassesFilter(proc *Process, filter ProcessFilter) bool {
	if filter.CurrentUserOnly {
		return proc.Owner == currentUser()
	}
	return true
}

// SetFingerprintFunc wires a handler's FingerprintParts method into the
// Process so that Fingerprint() delegates to it instead of the legacy switch.
// Called at the end of each handler's Enrich, on both cached and uncached paths.
func SetFingerprintFunc(proc *Process, fn func(*Process) []string) {
	proc.fingerprintParts = fn
}

// BuildServiceSetting constructs a ServiceSetting with all common fields.
// Handlers call this and then set any language-specific extras on the result.
func (b *BaseHandler) BuildServiceSetting(proc *Process) *ServiceSetting {
	langStr := b.languageString()
	key := "host-" + langStr + "-" + sanitize(proc.ServiceName)
	unitname := proc.DetailString(DetailSystemdUnit)
	serviceType := b.defaultServiceType()
	if unitname != "" {
		serviceType = "systemd"
	}

	serviceName := proc.ServiceName

	if proc.IsInContainer() {
		serviceType = "docker"
		if proc.ContainerInfo.ContainerID != "" && len(proc.ContainerInfo.ContainerID) >= 12 {
			key = b.containerKeyPrefix() + "-" + langStr + "-" + proc.ContainerInfo.ContainerID[:12]
			if b.Config.OverrideServiceNameOnContainer {
				serviceName = proc.ContainerInfo.ContainerName
			}
		}
	}

	agentType := deriveAgentType(proc.HasAgent, proc.AgentPath, proc.IsMiddlewareAgent)

	return &ServiceSetting{
		PID:               proc.PID,
		ServiceName:       serviceName,
		Owner:             proc.Owner,
		Status:            proc.Status,
		Enabled:           true,
		ServiceType:       serviceType,
		Language:          langStr,
		RuntimeVersion:    proc.RuntimeVersion,
		HasAgent:          proc.HasAgent,
		IsMiddlewareAgent: proc.IsMiddlewareAgent,
		AgentType:         agentType,
		AgentPath:         proc.AgentPath,
		Instrumented:      proc.HasAgent,
		Key:               key,
		SystemdUnit:       unitname,
		Listeners:         proc.Listeners(),
		Fingerprint:       proc.Fingerprint(),
		IntegrationType:   proc.IntegrationType(),
	}
}
