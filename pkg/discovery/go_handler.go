// go_handler.go implements the LanguageHandler interface for Go processes.
// It detects compiled Go binaries via embedded Go build info in the ELF
// header, enriches them with module path and working directory, and converts
// to ServiceSettings for backend reporting.
package discovery

import (
	"debug/buildinfo"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GoHandler implements LanguageHandler for Go processes.
type GoHandler struct{ BaseHandler }

func NewGoHandler() *GoHandler {
	return &GoHandler{BaseHandler: BaseHandler{Config: HandlerConfig{
		Language:                       LangGo,
		RuntimeName:                    "go",
		RuntimeDescription:             "Go Runtime",
		OverrideServiceNameOnContainer: true,
		CacheDetailMapping: []CacheDetailMap{
			{DetailGoModule, func(c ProcessCacheEntry) any { return c.EntryPoint }},
		},
	}}}
}

// Lang returns LangGo.
func (h *GoHandler) Lang() Language { return LangGo }

// Detect returns true if the executable is a compiled Go binary. Go embeds
// build info (module path, Go version) in the binary — reading it is the
// most reliable detection method since Go binaries have arbitrary exe names.
// Infrastructure binaries (dockerd, kubelet, etc.) are excluded.
//
// Uses /proc/<pid>/exe directly instead of the readlink result because
// containerized processes have exe paths inside their mount namespace
// (e.g. "/server") that don't exist on the host filesystem.
func (h *GoHandler) Detect(proc *ProcessInfo) bool {
	if isIgnoredGoBinary(proc.ExeName) {
		return false
	}
	_, err := buildinfo.ReadFile(fmt.Sprintf("/proc/%d/exe", proc.PID))
	return err == nil
}

// Enrich populates a Process struct with Go-specific details.
// Returns nil if the process should be skipped.
func (h *GoHandler) Enrich(info *ProcessInfo, opts DiscoveryOptions, detector *ContainerDetector) *Process {
	pid := info.PID

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

	proc := h.BuildProcess(info, owner, createTime, "unknown")

	if h.ApplyContainerInfo(proc, opts, detector) {
		return nil
	}

	h.extractGoInfo(proc, info)
	enrichCommonDetails(proc)
	h.extractServiceName(proc)
	h.detectInstrumentation(proc)

	entry := h.WriteCacheEntry(proc)
	entry.EntryPoint = proc.DetailString(DetailGoModule)
	CacheProcessMetadata(pid, alignedTime, entry)

	SetFingerprintFunc(proc, h.FingerprintParts)
	return proc
}

func (h *GoHandler) PassesFilter(proc *Process, filter ProcessFilter) bool {
	return h.DefaultPassesFilter(proc, filter)
}

func (h *GoHandler) ToServiceSetting(proc *Process) *ServiceSetting {
	return h.BuildServiceSetting(proc)
}

func (h *GoHandler) FingerprintParts(proc *Process) []string {
	var parts []string
	if mod := proc.DetailString(DetailGoModule); mod != "" {
		parts = append(parts, mod)
	}
	parts = append(parts, proc.ExecutablePath)
	return parts
}

// --- Private helpers ---

func (h *GoHandler) extractGoInfo(proc *Process, info *ProcessInfo) {
	bi, err := buildinfo.ReadFile(fmt.Sprintf("/proc/%d/exe", proc.PID))
	if err != nil {
		return
	}

	proc.RuntimeVersion = bi.GoVersion
	if bi.Main.Path != "" {
		proc.Details[DetailGoModule] = bi.Main.Path
	}

	if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", proc.PID)); err == nil {
		proc.Details[DetailWorkingDirectory] = cwd
	}
}

// extractServiceName determines the service name for a Go process.
// Priority: 1. Container name → 2. OTEL_SERVICE_NAME/SERVICE_NAME env →
// 3. Systemd unit → 4. Executable name → 5. Module path last segment →
// 6. Working directory → 7. "go-service"
func (h *GoHandler) extractServiceName(proc *Process) {
	if proc.IsInContainer() && proc.ContainerInfo.ContainerName != "" {
		proc.ServiceName = cleanName(proc.ContainerInfo.ContainerName)
		if proc.ServiceName != "" {
			return
		}
	}

	if name := extractServiceNameFromEnviron(proc.PID); name != "" {
		proc.ServiceName = name
		return
	}

	if unitName := extractSystemdUnit(proc.PID); unitName != "" {
		proc.ServiceName = cleanName(unitName)
		if proc.ServiceName != "" {
			return
		}
	}

	if name := cleanName(proc.ExecutableName); name != "" {
		proc.ServiceName = name
		return
	}

	if modPath := proc.DetailString(DetailGoModule); modPath != "" {
		parts := strings.Split(modPath, "/")
		if name := cleanName(parts[len(parts)-1]); name != "" {
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

	proc.ServiceName = "go-service"
}

// detectInstrumentation checks for OTel eBPF or LD_PRELOAD-based instrumentation.
func (h *GoHandler) detectInstrumentation(proc *Process) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", proc.PID))
	if err != nil {
		return
	}

	for _, env := range strings.Split(string(data), "\x00") {
		if strings.HasPrefix(env, "LD_PRELOAD=") {
			ldPreload := strings.TrimPrefix(env, "LD_PRELOAD=")
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

// goIgnoredBinaries lists exact-match names for known infrastructure binaries.
var goIgnoredBinaries = map[string]bool{
	"dockerd":         true,
	"docker-proxy":    true,
	"containerd":      true,
	"kubelet":         true,
	"kube-proxy":      true,
	"kube-apiserver":  true,
	"kube-scheduler":  true,
	"kube-controller": true,
	"etcd":            true,
	"coredns":         true,
	"cri-dockerd":     true,
	"buildkitd":       true,
	"runc":            true,
	"cni":             true,
	"flannel":         true,
	"calico-node":     true,
	"prometheus":      true,
	"alertmanager":    true,
	"grafana":         true,
	"consul":          true,
	"vault":           true,
	"terraform":       true,
	"mw-agent":        true,
	"mw-host-agent":   true,
	"obi":             true,
	"obi-agent":       true,
	"otel-collector":  true,
	"otelcol":         true,
	"otelcol-contrib": true,
	"mw-injector":     true,
	"esbuild":         true,
	"gopls":           true,
	"dlv":             true,
	"staticcheck":     true,
	"golangci-lint":   true,
	"lazydocker":      true,
	"lazygit":         true,
	"vanta-agent":     true,
}

// goIgnoredPrefixes matches binaries whose names vary with versions or
// runtime variants (e.g. containerd-shim-runc-v2, gopls_0.21.1_go_1.26.2).
var goIgnoredPrefixes = []string{
	"containerd-shim",
	"kube-",
	"vanta",
	"gopls",
}

// isIgnoredGoBinary returns true if the executable name is a known
// infrastructure/tooling binary that should not be discovered as a user service.
func isIgnoredGoBinary(exeName string) bool {
	name := strings.ToLower(filepath.Base(exeName))
	if goIgnoredBinaries[name] {
		return true
	}
	for _, prefix := range goIgnoredPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
