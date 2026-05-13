// python_handler.go implements the LanguageHandler interface for Python processes.
// It handles detection of CPython/PyPy processes and Python-based binaries
// (gunicorn, uvicorn, celery), enrichment with module/entry point info,
// process manager details, and instrumentation state. Also contains Python-specific
// helper functions and agent type definitions.
package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PythonHandler implements LanguageHandler for Python processes.
// It detects CPython/PyPy processes and Python-based binaries (gunicorn,
// uvicorn, celery), enriches them with module/entry point info, process
// manager details, and instrumentation state.
type PythonHandler struct{ BaseHandler }

func NewPythonHandler() *PythonHandler {
	return &PythonHandler{BaseHandler: BaseHandler{Config: HandlerConfig{
		Language:           LangPython,
		RuntimeName:        "python",
		RuntimeDescription: "Python Interpreter",
		ContainerKeyPrefix: "docker",
		CacheDetailMapping: []CacheDetailMap{
			{DetailEntryPoint, func(c ProcessCacheEntry) any { return c.EntryPoint }},
			{DetailProcessManager, func(c ProcessCacheEntry) any { return c.ServiceType }},
			{DetailIsGunicorn, func(c ProcessCacheEntry) any { return c.ServiceType == "gunicorn" }},
			{DetailIsUvicorn, func(c ProcessCacheEntry) any { return c.ServiceType == "uvicorn" }},
			{DetailIsCelery, func(c ProcessCacheEntry) any { return c.ServiceType == "celery" }},
			{DetailModulePath, func(c ProcessCacheEntry) any { return c.ModulePath }},
		},
		ExtraCacheWriter: func(proc *Process, entry *ProcessCacheEntry) {
			entry.ModulePath = proc.DetailString(DetailModulePath)
		},
	}}}
}

// Lang returns LangPython.
func (h *PythonHandler) Lang() Language { return LangPython }

// Detect returns true if the process is a Python process, identified by
// executable name (python, python3, pypy), Python-based binaries
// (gunicorn, uvicorn, celery, flask), or .py file references in cmdline.
func (h *PythonHandler) Detect(proc *ProcessInfo) bool {
	exeLower := strings.ToLower(proc.ExeName)

	if pythonExecutables[exeLower] || strings.HasPrefix(exeLower, "python3.") {
		return true
	}

	if pythonBinaries[exeLower] {
		return true
	}

	cmdLower := strings.ToLower(proc.CmdLine)
	for _, pattern := range pythonCmdPatterns {
		if strings.Contains(cmdLower, pattern) {
			return true
		}
	}

	// Fallback: any .py file reference in cmdline
	if strings.Contains(cmdLower, ".py") {
		return true
	}

	return false
}

// Enrich populates a Process struct with Python-specific details.
// Returns nil if the process should be skipped.
func (h *PythonHandler) Enrich(info *ProcessInfo, opts DiscoveryOptions, detector *ContainerDetector) *Process {
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

	isSubProcess := strings.Contains(info.CmdLine, "multiprocessing.spawn") ||
		strings.Contains(info.CmdLine, "resource_tracker")
	if isSubProcess {
		CacheProcessMetadata(pid, alignedTime, ProcessCacheEntry{Ignore: true, Owner: owner})
		return nil
	}

	h.extractPythonInfo(proc, cmdArgs)
	enrichCommonDetails(proc)
	h.extractServiceName(proc, cmdArgs)
	h.detectProcessManager(proc, cmdArgs)
	h.detectInstrumentation(proc, cmdArgs)

	CacheProcessMetadata(pid, alignedTime, h.WriteCacheEntry(proc))

	SetFingerprintFunc(proc, h.FingerprintParts)
	return proc
}

func (h *PythonHandler) PassesFilter(proc *Process, filter ProcessFilter) bool {
	return h.DefaultPassesFilter(proc, filter)
}

func (h *PythonHandler) ToServiceSetting(proc *Process) *ServiceSetting {
	ss := h.BuildServiceSetting(proc)

	ss.MainClass = proc.DetailString(DetailModulePath)
	ss.JarFile = proc.DetailString(DetailEntryPoint)
	ss.ProcessManager = proc.DetailString(DetailProcessManager)

	if !proc.IsInContainer() && proc.DetailBool(DetailIsCelery) {
		ss.ServiceType = "worker"
	}

	return ss
}

func (h *PythonHandler) FingerprintParts(proc *Process) []string {
	var parts []string
	if mp := proc.DetailString(DetailModulePath); mp != "" {
		parts = append(parts, mp)
	}
	if ep := proc.DetailString(DetailEntryPoint); ep != "" {
		parts = append(parts, ep)
	}
	return parts
}

// --- Private helpers ---

func (h *PythonHandler) extractPythonInfo(proc *Process, cmdArgs []string) {
	if strings.Contains(proc.ExecutablePath, "/bin/python") {
		proc.Details[DetailVenvPath] = filepath.Dir(filepath.Dir(proc.ExecutablePath))
	}

	if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", proc.PID)); err == nil {
		proc.Details[DetailWorkingDirectory] = cwd
	}

	for i, arg := range cmdArgs {
		if i == 0 {
			continue
		}

		if arg == "-m" && i+1 < len(cmdArgs) {
			mod := cmdArgs[i+1]
			proc.Details[DetailModulePath] = mod
			proc.Details[DetailEntryPoint] = mod
			return
		}

		if strings.HasPrefix(arg, "-") {
			continue
		}

		// Skip known Python tool names — the next positional arg is the
		// real entry point (e.g. "uvicorn main:app" → entry point is "main:app").
		argBase := filepath.Base(arg)
		if pythonBinaries[strings.ToLower(argBase)] {
			continue
		}

		proc.Details[DetailEntryPoint] = filepath.Base(arg)

		if strings.HasSuffix(arg, ".py") {
			if !filepath.IsAbs(arg) {
				if cwd := proc.DetailString(DetailWorkingDirectory); cwd != "" {
					proc.Details[DetailWorkingDirectory] = filepath.Dir(filepath.Join(cwd, arg))
				}
			} else {
				proc.Details[DetailWorkingDirectory] = filepath.Dir(arg)
			}
		}
		return
	}
}

func (h *PythonHandler) extractServiceName(proc *Process, cmdArgs []string) {
	// Level 1: Explicit Environment
	if name := extractServiceNameFromEnviron(proc.PID); name != "" {
		proc.ServiceName = cleanName(name)
		return
	}

	// Level 2: Container name
	if proc.ContainerInfo != nil && proc.ContainerInfo.IsContainer {
		if proc.ContainerInfo.ContainerName != "" {
			proc.ServiceName = proc.ContainerInfo.ContainerName
			return
		}
	}

	// Level 3: VirtualEnv path analysis
	for _, arg := range proc.CommandArgs {
		if strings.Contains(arg, "/.venv/") || strings.Contains(arg, "/venv/") || strings.Contains(arg, "/.env/") {
			cleanPath := filepath.Clean(arg)
			parts := strings.Split(cleanPath, string(os.PathSeparator))

			for i, part := range parts {
				if part == ".venv" || part == "venv" || part == ".env" {
					if i > 0 && !isGenericPython(parts[i-1]) {
						proc.ServiceName = parts[i-1]
						return
					}
				}
			}
		}
	}

	// Level 4: Entry point / module analysis
	for i, arg := range cmdArgs {
		if strings.Contains(arg, ":") && !strings.Contains(arg, "/") {
			modName := strings.Split(arg, ":")[0]
			if !isGenericPython(modName) {
				proc.ServiceName = modName
				return
			}
		}

		if strings.HasSuffix(arg, ".py") || (i > 0 && !strings.HasPrefix(arg, "-") && strings.Contains(arg, "/")) {
			baseName := filepath.Base(arg)
			baseName = strings.TrimSuffix(baseName, ".py")
			if !isGenericPython(baseName) {
				proc.ServiceName = baseName
				return
			}
		}
	}

	// Level 5: Absolute script path analysis
	entryPoint := proc.DetailString(DetailEntryPoint)
	if entryPoint != "" {
		fullPath := entryPoint
		workDir := proc.DetailString(DetailWorkingDirectory)
		if !filepath.IsAbs(fullPath) && workDir != "" {
			fullPath = filepath.Join(workDir, fullPath)
		}

		parentDir := filepath.Dir(fullPath)
		dirName := filepath.Base(parentDir)

		if !isGenericPython(dirName) && dirName != "." && dirName != "/" {
			proc.ServiceName = dirName
			return
		}
	}

	// Level 6: Working directory fallback
	workDir := proc.DetailString(DetailWorkingDirectory)
	if workDir != "" {
		if name := serviceNameFromWorkDir(workDir); name != "" {
			proc.ServiceName = name
			return
		}
	}

	proc.ServiceName = "python-service"
}

func (h *PythonHandler) detectProcessManager(proc *Process, cmdArgs []string) {
	cmdlineLower := strings.ToLower(strings.Join(cmdArgs, " "))

	if strings.Contains(cmdlineLower, "gunicorn") {
		proc.Details[DetailIsGunicorn] = true
		proc.Details[DetailProcessManager] = "gunicorn"
	} else if strings.Contains(cmdlineLower, "uvicorn") {
		proc.Details[DetailIsUvicorn] = true
		proc.Details[DetailProcessManager] = "uvicorn"
	} else if strings.Contains(cmdlineLower, "celery") {
		proc.Details[DetailIsCelery] = true
		proc.Details[DetailProcessManager] = "celery"
	}
}

func (h *PythonHandler) detectInstrumentation(proc *Process, cmdArgs []string) {
	cmdline := strings.Join(cmdArgs, " ")

	if strings.Contains(cmdline, "opentelemetry-instrument") {
		proc.HasAgent = true
		proc.AgentType = PythonAgentOpenTelemetry.String()
	}

	environPath := fmt.Sprintf("/proc/%d/environ", proc.PID)
	data, err := os.ReadFile(environPath)
	if err != nil {
		return
	}
	env := string(data)

	if strings.Contains(env, "PYTHONPATH") && strings.Contains(env, "mw_bootstrap") {
		proc.HasAgent = true
		proc.IsMiddlewareAgent = true
		proc.AgentType = PythonAgentMiddleware.String()
	}

	if strings.Contains(env, "PYTHON_AUTO_INSTRUMENTATION_AGENT_PATH_PREFIX=") &&
		strings.Contains(env, "LD_PRELOAD="+defaultLibOtelInjectorPath) {
		proc.HasAgent = true
		proc.AgentType = PythonAgentOtelInjector.String()
		proc.AgentPath = defaultPythonAgentBasePath
	}
}

// --- Python-specific lookup tables, types, and helpers ---

// pythonExecutables lists executable names that identify a Python process.
var pythonExecutables = map[string]bool{
	"python":  true,
	"python2": true,
	"python3": true,
	"pypy":    true,
	"pypy3":   true,
}

// pythonBinaries lists Python-based binary names (e.g. web servers, task queues)
// that should be treated as Python processes.
var pythonBinaries = map[string]bool{
	"gunicorn":     true,
	"uvicorn":      true,
	"celery":       true,
	"flask":        true,
	"django-admin": true,
}

// pythonCmdPatterns lists command line patterns that indicate a Python process.
var pythonCmdPatterns = []string{
	"python ",
	"python3 ",
	"gunicorn ",
	"uvicorn ",
	"celery ",
	"manage.py runserver",
	"flask run",
}

// PythonAgentType represents the type of Python instrumentation agent detected.
type PythonAgentType int

const (
	PythonAgentNone          PythonAgentType = iota
	PythonAgentOpenTelemetry                 // opentelemetry-instrument wrapper
	PythonAgentMiddleware                    // mw_bootstrap injected into PYTHONPATH
	PythonAgentOtelInjector                  // LD_PRELOAD + libotelinject.so drop-in
	PythonAgentOther
)

// String returns a human-readable label for the Python agent type.
func (a PythonAgentType) String() string {
	switch a {
	case PythonAgentNone:
		return "none"
	case PythonAgentOpenTelemetry:
		return "opentelemetry"
	case PythonAgentMiddleware:
		return "middleware"
	case PythonAgentOtelInjector:
		return "otel-injector"
	case PythonAgentOther:
		return "other"
	default:
		return "unknown"
	}
}

const (
	// defaultLibOtelInjectorPath is the default LD_PRELOAD path for the
	// OpenTelemetry injector shared library.
	defaultLibOtelInjectorPath = "/usr/lib/opentelemetry/libotelinject.so"
	// defaultPythonAgentBasePath is the default base path for the Python
	// OpenTelemetry agent installation.
	defaultPythonAgentBasePath = "/opt/otel-python-agent"
)

// isGenericPython returns true if the name is too generic to be useful as
// a Python service name (e.g. "app", "main", "server", "python").
func isGenericPython(name string) bool {
	generics := map[string]bool{
		"app": true, "main": true, "server": true, "index": true,
		"python": true, "python3": true, "uvicorn": true, "gunicorn": true,
		"bin": true, "src": true, "lib": true,
	}
	return generics[strings.ToLower(name)]
}
