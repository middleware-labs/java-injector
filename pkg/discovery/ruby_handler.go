package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RubyHandler implements LanguageHandler for Ruby processes.
type RubyHandler struct{ BaseHandler }

func NewRubyHandler() *RubyHandler {
	return &RubyHandler{BaseHandler: BaseHandler{Config: HandlerConfig{
		Language:                       LangRuby,
		RuntimeName:                    "ruby",
		RuntimeDescription:             "Ruby Interpreter",
		OverrideServiceNameOnContainer: true,
		CacheDetailMapping: []CacheDetailMap{
			{DetailEntryPoint, func(c ProcessCacheEntry) any { return c.EntryPoint }},
			{DetailProcessManager, func(c ProcessCacheEntry) any { return c.ServiceType }},
		},
	}}}
}

func (h *RubyHandler) Lang() Language { return LangRuby }

// Detect returns true if the process is a Ruby process, identified by
// executable name (ruby, jruby), Ruby-based binaries (rails, puma,
// sidekiq, unicorn, rake, bundler), or .rb file references in cmdline.
func (h *RubyHandler) Detect(proc *ProcessInfo) bool {
	exeLower := strings.ToLower(proc.ExeName)

	if rubyExecutables[exeLower] {
		return true
	}

	if rubyBinaries[exeLower] {
		return true
	}

	cmdLower := strings.ToLower(proc.CmdLine)
	for _, pattern := range rubyCmdPatterns {
		if strings.Contains(cmdLower, pattern) {
			return true
		}
	}

	if strings.Contains(cmdLower, ".rb") {
		return true
	}

	return false
}

func (h *RubyHandler) Enrich(info *ProcessInfo, opts DiscoveryOptions, detector *ContainerDetector) *Process {
	pid := info.PID
	cmdArgs := info.CmdArgs

	owner := readProcessOwner(pid)
	createTime := readProcessCreateTime(pid)
	alignedTime := (createTime / 1000) * 1000

	if cached, hit := GetCachedProcessMetadata(pid, alignedTime); hit {
		if cached.Ignore {
			return nil
		}
		proc := h.BuildProcessFromCache(info, cached, createTime)
		SetFingerprintFunc(proc, h.FingerprintParts)
		return proc
	}

	if isIgnoredSystemdUnit(pid) {
		CacheProcessMetadata(pid, alignedTime, ProcessCacheEntry{Ignore: true})
		return nil
	}

	proc := h.BuildProcess(info, owner, createTime, "")

	if h.ApplyContainerInfo(proc, opts, detector) {
		return nil
	}

	h.extractRubyInfo(proc, cmdArgs)
	enrichCommonDetails(proc)
	h.extractServiceName(proc, cmdArgs)
	h.detectProcessManager(proc, cmdArgs)
	h.detectInstrumentation(proc)

	CacheProcessMetadata(pid, alignedTime, h.WriteCacheEntry(proc))

	SetFingerprintFunc(proc, h.FingerprintParts)
	return proc
}

func (h *RubyHandler) PassesFilter(proc *Process, filter ProcessFilter) bool {
	return h.DefaultPassesFilter(proc, filter)
}

func (h *RubyHandler) ToServiceSetting(proc *Process) *ServiceSetting {
	ss := h.BuildServiceSetting(proc)
	ss.ProcessManager = proc.DetailString(DetailProcessManager)
	return ss
}

func (h *RubyHandler) FingerprintParts(proc *Process) []string {
	var parts []string
	if ep := proc.DetailString(DetailEntryPoint); ep != "" {
		parts = append(parts, ep)
	}
	parts = append(parts, proc.ExecutablePath)
	return parts
}

// --- Private helpers ---

func (h *RubyHandler) extractRubyInfo(proc *Process, cmdArgs []string) {
	if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", proc.PID)); err == nil {
		proc.Details[DetailWorkingDirectory] = cwd
	}

	// Extract Ruby version from exe path (e.g. /usr/bin/ruby3.2 → "3.2")
	exeBase := filepath.Base(proc.ExecutablePath)
	if strings.HasPrefix(exeBase, "ruby") {
		ver := strings.TrimPrefix(exeBase, "ruby")
		if ver != "" {
			proc.RuntimeVersion = ver
		}
	}

	for i, arg := range cmdArgs {
		if i == 0 {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}

		argBase := filepath.Base(arg)
		if rubyBinaries[strings.ToLower(argBase)] {
			continue
		}

		proc.Details[DetailEntryPoint] = filepath.Base(arg)
		return
	}
}

// extractServiceName determines the service name for a Ruby process.
// Priority: 1. OTEL_SERVICE_NAME/SERVICE_NAME env → 2. Container name →
// 3. Systemd unit → 4. Gemfile app name → 5. Entry point basename →
// 6. Working directory → 7. "ruby-service"
func (h *RubyHandler) extractServiceName(proc *Process, cmdArgs []string) {
	if name := extractServiceNameFromEnviron(proc.PID); name != "" {
		proc.ServiceName = cleanName(name)
		if proc.ServiceName != "" {
			return
		}
	}

	if proc.IsInContainer() && proc.ContainerInfo.ContainerName != "" {
		proc.ServiceName = cleanName(proc.ContainerInfo.ContainerName)
		if proc.ServiceName != "" {
			return
		}
	}

	if unitName := extractSystemdUnit(proc.PID); unitName != "" {
		proc.ServiceName = cleanName(unitName)
		if proc.ServiceName != "" {
			return
		}
	}

	// Ruby framework binaries (puma, sidekiq, unicorn) have meaningful exe names
	if rubyBinaries[strings.ToLower(proc.ExecutableName)] {
		if name := cleanName(proc.ExecutableName); name != "" {
			proc.ServiceName = name
			return
		}
	}

	// Try Gemfile-based app name from working directory
	if dir := proc.DetailString(DetailWorkingDirectory); dir != "" {
		if name := rubyAppNameFromGemfile(dir); name != "" {
			proc.ServiceName = name
			return
		}
	}

	// Rails app name from cmdline (e.g. "rails server" → use working dir name)
	for _, arg := range cmdArgs {
		if strings.Contains(arg, "config.ru") || strings.Contains(arg, "rails") {
			if dir := proc.DetailString(DetailWorkingDirectory); dir != "" {
				if name := serviceNameFromWorkDir(dir); name != "" {
					proc.ServiceName = name
					return
				}
			}
		}
	}

	if ep := proc.DetailString(DetailEntryPoint); ep != "" {
		base := strings.TrimSuffix(filepath.Base(ep), ".rb")
		if name := cleanName(base); name != "" {
			proc.ServiceName = name
			return
		}
	}

	if dir := proc.DetailString(DetailWorkingDirectory); dir != "" {
		if name := serviceNameFromWorkDir(dir); name != "" {
			proc.ServiceName = name
			return
		}
	}

	proc.ServiceName = "ruby-service"
}

func (h *RubyHandler) detectProcessManager(proc *Process, cmdArgs []string) {
	cmdlineLower := strings.ToLower(strings.Join(cmdArgs, " "))

	if strings.Contains(cmdlineLower, "puma") {
		proc.Details[DetailProcessManager] = "puma"
	} else if strings.Contains(cmdlineLower, "unicorn") {
		proc.Details[DetailProcessManager] = "unicorn"
	} else if strings.Contains(cmdlineLower, "sidekiq") {
		proc.Details[DetailProcessManager] = "sidekiq"
	} else if strings.Contains(cmdlineLower, "passenger") {
		proc.Details[DetailProcessManager] = "passenger"
	}
}

func (h *RubyHandler) detectInstrumentation(proc *Process) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", proc.PID))
	if err != nil {
		return
	}

	env := string(data)

	// OpenTelemetry Ruby SDK sets OTEL_TRACES_EXPORTER or is loaded via RUBYOPT
	if strings.Contains(env, "OTEL_TRACES_EXPORTER=") || strings.Contains(env, "opentelemetry") {
		proc.HasAgent = true
		proc.AgentType = "opentelemetry"
	}

	for _, e := range strings.Split(env, "\x00") {
		if strings.HasPrefix(e, "LD_PRELOAD=") {
			ldPreload := strings.TrimPrefix(e, "LD_PRELOAD=")
			if path := extractLibOtelInjectPath(ldPreload); path != "" {
				proc.HasAgent = true
				proc.AgentPath = path
				proc.AgentType = "otel-injector"
				if strings.Contains(path, "middleware") {
					proc.IsMiddlewareAgent = true
				}
			}
		}
	}
}

// rubyAppNameFromGemfile attempts to read a Gemfile in the working directory
// and infer the app name. If there's no Gemfile, returns "".
// We don't parse the Gemfile itself — just use the directory name.
func rubyAppNameFromGemfile(dir string) string {
	gemfilePath := filepath.Join(dir, "Gemfile")
	if _, err := os.Stat(gemfilePath); err != nil {
		return ""
	}
	return cleanName(filepath.Base(dir))
}

// --- Ruby-specific lookup tables ---

var rubyExecutables = map[string]bool{
	"ruby":  true,
	"jruby": true,
	"truffleruby": true,
}

var rubyBinaries = map[string]bool{
	"rails":     true,
	"rake":      true,
	"puma":      true,
	"sidekiq":   true,
	"unicorn":   true,
	"passenger": true,
	"bundler":   true,
	"bundle":    true,
	"resque":    true,
	"thin":      true,
	"rackup":    true,
}

var rubyCmdPatterns = []string{
	"ruby ",
	"rails ",
	"puma ",
	"sidekiq ",
	"unicorn ",
	"bundle exec",
	"passenger ",
	"rackup ",
}
