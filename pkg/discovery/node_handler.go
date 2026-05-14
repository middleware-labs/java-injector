// node_handler.go implements the LanguageHandler interface for Node.js processes.
// It handles detection of Node/npm/yarn processes, enrichment with entry point,
// package info, process manager details, and instrumentation state. Also contains
// Node-specific helper functions for executable matching and agent detection.
package discovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// NodeHandler implements LanguageHandler for Node.js processes.
// It detects Node/npm/yarn processes, enriches them with entry point,
// package info, process manager details, and instrumentation state.
type NodeHandler struct{ BaseHandler }

func NewNodeHandler() *NodeHandler {
	return &NodeHandler{BaseHandler: BaseHandler{Config: HandlerConfig{
		Language:                       LangNode,
		RuntimeName:                    "node",
		RuntimeDescription:             "Node.js Runtime",
		OverrideServiceNameOnContainer: true,
		CacheDetailMapping: []CacheDetailMap{
			{DetailEntryPoint, func(c ProcessCacheEntry) any { return c.EntryPoint }},
			{DetailProcessManager, func(c ProcessCacheEntry) any { return c.ServiceType }},
			{DetailIsPM2, func(c ProcessCacheEntry) any { return c.ServiceType == "pm2" }},
			{DetailIsForever, func(c ProcessCacheEntry) any { return c.ServiceType == "forever" }},
			{DetailPackageName, func(c ProcessCacheEntry) any { return c.PackageName }},
		},
		ExtraCacheWriter: func(proc *Process, entry *ProcessCacheEntry) {
			entry.PackageName = proc.DetailString(DetailPackageName)
		},
	}}}
}

// Lang returns LangNode.
func (h *NodeHandler) Lang() Language { return LangNode }

// Detect returns true if the process is a Node.js process, identified
// by executable name (node, nodejs). Package managers (npm, yarn, etc.)
// resolve to the node binary via symlink/shebang, so ExeName is always
// "node" — no cmdline pattern matching needed.
func (h *NodeHandler) Detect(proc *ProcessInfo) bool {
	return nodeExecutables[strings.ToLower(proc.ExeName)]
}

// Enrich populates a Process struct with Node.js-specific details.
// Returns nil if the process should be skipped.
func (h *NodeHandler) Enrich(info *ProcessInfo, opts DiscoveryOptions, detector *ContainerDetector) *Process {
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

	if isNodeLauncher(cmdArgs) {
		CacheProcessMetadata(pid, alignedTime, ProcessCacheEntry{Ignore: true})
		return nil
	}

	proc := h.BuildProcess(info, owner, createTime, "unknown")

	if h.ApplyContainerInfo(proc, opts, detector) {
		return nil
	}

	h.extractNodeInfo(proc, cmdArgs)
	enrichCommonDetails(proc)
	h.extractServiceName(proc, cmdArgs)
	h.detectProcessManager(proc, cmdArgs)
	h.detectInstrumentation(proc, cmdArgs)

	CacheProcessMetadata(pid, alignedTime, h.WriteCacheEntry(proc))

	SetFingerprintFunc(proc, h.FingerprintParts)
	return proc
}


func (h *NodeHandler) ToServiceSetting(proc *Process) *ServiceSetting {
	return h.BuildServiceSetting(proc)
}

func (h *NodeHandler) FingerprintParts(proc *Process) []string {
	var parts []string
	if pkg := proc.DetailString(DetailPackageName); pkg != "" && pkg != "unknown" {
		parts = append(parts, pkg)
	}
	if ep := proc.DetailString(DetailEntryPoint); ep != "" {
		parts = append(parts, ep)
	}
	return parts
}

// --- Private helpers ---

func (h *NodeHandler) extractNodeInfo(proc *Process, cmdArgs []string) {
	var entryPoint, workingDirectory string

	// Read the target process's working directory from /proc, not the agent's own cwd.
	if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", proc.PID)); err == nil {
		workingDirectory = cwd
	}

	for i, arg := range cmdArgs {
		if i == 0 || strings.HasPrefix(arg, "-") {
			continue
		}

		// First positional arg after "node" is the entry point — whether
		// it's "index.js", "server.mjs", or an extensionless script like
		// "codegraph" (npm-installed CLI with #!/usr/bin/env node shebang).
		entryPoint = arg

		if filepath.IsAbs(entryPoint) {
			workingDirectory = filepath.Dir(entryPoint)
			entryPoint = filepath.Base(entryPoint)
		} else if workingDirectory != "" {
			absPath := filepath.Join(workingDirectory, entryPoint)
			workingDirectory = filepath.Dir(absPath)
			entryPoint = filepath.Base(absPath)
		}
		break
	}

	proc.Details[DetailEntryPoint] = entryPoint
	proc.Details[DetailWorkingDirectory] = workingDirectory

	if workingDirectory != "" {
		packageJsonPath := filepath.Join(workingDirectory, "package.json")
		proc.Details[DetailPackageJsonPath] = packageJsonPath

		// Skip reading package.json for containerized processes — the path
		// is inside the container's mount namespace and doesn't exist on the
		// host. Container name already provides identity via ContainerInfo.
		if !proc.IsInContainer() {
			if data, err := os.ReadFile(packageJsonPath); err == nil {
				var pkg struct {
					Name    string `json:"name"`
					Version string `json:"version"`
				}
				if json.Unmarshal(data, &pkg) == nil {
					if pkg.Name != "" {
						proc.Details[DetailPackageName] = pkg.Name
					}
					if pkg.Version != "" {
						proc.Details[DetailPackageVersion] = pkg.Version
					}
				}
			}
		}
	}
}

func (h *NodeHandler) extractServiceName(proc *Process, cmdArgs []string) {
	if unitName := extractSystemdUnit(proc.PID); unitName != "" {
		proc.ServiceName = cleanName(unitName)
		return
	}

	if name := extractNodeServiceNameFromCmdArgs(cmdArgs); name != "" {
		proc.ServiceName = name
		return
	}

	pkgName := proc.DetailString(DetailPackageName)
	if pkgName != "" && pkgName != "unknown" {
		if name := cleanName(pkgName); name != "" {
			proc.ServiceName = name
			return
		}
	}

	entryPoint := proc.DetailString(DetailEntryPoint)
	if entryPoint != "" {
		if name := extractNameFromNodeScript(entryPoint); name != "" {
			proc.ServiceName = name
			return
		}
	}

	workDir := proc.DetailString(DetailWorkingDirectory)
	if workDir != "" {
		if name := serviceNameFromWorkDir(workDir); name != "" {
			proc.ServiceName = name
			return
		}
	}

	proc.ServiceName = "node-service"
}

func (h *NodeHandler) detectProcessManager(proc *Process, cmdArgs []string) {
	cmdline := strings.ToLower(strings.Join(cmdArgs, " "))

	if strings.Contains(cmdline, "pm2") || strings.Contains(cmdline, "/pm2/") {
		proc.Details[DetailIsPM2] = true
		proc.Details[DetailProcessManager] = "pm2"

		for _, arg := range cmdArgs {
			if strings.HasPrefix(arg, "--name=") {
				proc.Details[DetailPM2Name] = strings.TrimPrefix(arg, "--name=")
				break
			}
		}
	}

	if strings.Contains(cmdline, "forever") || strings.Contains(cmdline, "/forever/") {
		proc.Details[DetailIsForever] = true
		proc.Details[DetailProcessManager] = "forever"
	}

	if proc.DetailString(DetailProcessManager) == "" && proc.Owner != "root" && proc.Owner != "node" {
		proc.Details[DetailProcessManager] = "systemd"
	}
}

func (h *NodeHandler) detectInstrumentation(proc *Process, cmdArgs []string) {
	cmdline := strings.Join(cmdArgs, " ")

	if h.detectAgentInCmdline(proc, cmdline) {
		return
	}

	h.checkEnvForAgent(proc)
}

func (h *NodeHandler) detectAgentInCmdline(proc *Process, cmdline string) bool {
	agentPatterns := []string{"--require ", "-r ", "--import ", "--loader "}

	cmdlineLower := strings.ToLower(cmdline)
	for _, pattern := range agentPatterns {
		if strings.Contains(cmdlineLower, pattern) {
			agentPath := extractNodeAgentPathFromCmdline(cmdline, pattern)
			if agentPath != "" {
				h.setAgentInfo(proc, agentPath)
				return true
			}
		}
	}
	return false
}

func (h *NodeHandler) checkEnvForAgent(proc *Process) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", proc.PID))
	if err != nil {
		return
	}

	envVars := strings.Split(string(data), "\x00")
	for _, env := range envVars {
		if strings.HasPrefix(env, "NODE_OPTIONS=") {
			value := strings.TrimPrefix(env, "NODE_OPTIONS=")
			if agentPath := extractNodeAgentFromEnvValue(value); agentPath != "" {
				h.setAgentInfo(proc, agentPath)
				return
			}
		}

		if strings.HasPrefix(env, "LD_PRELOAD=") {
			value := strings.TrimPrefix(env, "LD_PRELOAD=")
			if path := extractLibOtelInjectPath(value); path != "" {
				proc.HasAgent = true
				proc.IsMiddlewareAgent = false
				proc.AgentPath = path
				proc.AgentType = "opentelemetry"
				return
			}
		}
	}
}

func (h *NodeHandler) setAgentInfo(proc *Process, agentPath string) {
	proc.HasAgent = true
	proc.AgentPath = agentPath

	mwPatterns := []string{"middleware", "mw-", "mw.js", "middleware-agent", "mw-register"}
	lower := strings.ToLower(agentPath)
	for _, p := range mwPatterns {
		if strings.Contains(lower, p) {
			proc.IsMiddlewareAgent = true
			proc.AgentType = "middleware"
			return
		}
	}
	proc.AgentType = "opentelemetry"
}

// --- Node.js-specific lookup tables and helpers ---

// nodeExecutables lists executable names that identify a Node.js process.
var nodeExecutables = map[string]bool{
	"node":   true,
	"nodejs": true,
}

// nodeLaunchers lists argv[0] values that identify Node.js infrastructure
// processes (package managers, process managers) rather than application
// processes. PM2's God Daemon rewrites its argv[0] to
// "PM2 v<version>: God Daemon (...)", so the first word is "pm2".
var nodeLaunchers = map[string]bool{
	"npm":      true,
	"npx":      true,
	"yarn":     true,
	"pnpm":     true,
	"corepack": true,
	"pm2":      true,
}

// pm2Binaries lists executable basenames for PM2 entry points. When node
// runs one of these as a script (e.g. "node /usr/local/bin/pm2-runtime"),
// the process is a PM2 daemon — not an application. Its listen ports flow
// to workers via InheritParentPorts.
var pm2Binaries = map[string]bool{
	"pm2":         true,
	"pm2-runtime": true,
	"pm2-dev":     true,
}

// isNodeLauncher returns true if the process is a Node.js package manager
// launcher (npm start, yarn run, etc.) rather than an actual application.
func isNodeLauncher(cmdArgs []string) bool {
	if len(cmdArgs) == 0 {
		return false
	}
	// When launched via "sh -c npm start", /proc/<pid>/cmdline may contain
	// "npm start" as a single space-joined arg instead of null-separated.
	// Extract the first word to handle both forms.
	first := strings.ToLower(cmdArgs[0])
	if i := strings.IndexByte(first, ' '); i > 0 {
		first = first[:i]
	}
	if nodeLaunchers[first] {
		return true
	}

	// When node runs a launcher script (e.g. "node /usr/local/bin/pm2-runtime"),
	// cmdArgs[0] is "node" which passes the check above. Detect PM2 daemon
	// processes by checking the first positional argument's basename.
	for _, arg := range cmdArgs[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		base := strings.ToLower(filepath.Base(arg))
		return pm2Binaries[base]
	}

	return false
}

// extractNodeServiceNameFromCmdArgs extracts a service name from Node.js
// command arguments by looking for --name=, --service=, or SERVICE_NAME= flags.
func extractNodeServiceNameFromCmdArgs(cmdArgs []string) string {
	serviceProperties := []string{"--name=", "--service=", "SERVICE_NAME=", "NODE_ENV="}

	for _, arg := range cmdArgs {
		for _, prop := range serviceProperties {
			if strings.Contains(arg, prop) {
				name := strings.TrimPrefix(arg, prop)
				name = strings.Trim(name, `"'`)
				if name != "" && name != "production" && name != "development" {
					return cleanName(name)
				}
			}
		}
	}
	return ""
}

// extractNameFromNodeScript derives a service name from a Node script filename,
// filtering out generic names like index.js, app.js, server.js.
func extractNameFromNodeScript(scriptName string) string {
	if scriptName == "" {
		return ""
	}

	baseName := strings.TrimSuffix(scriptName, filepath.Ext(scriptName))
	baseName = filepath.Base(baseName)

	genericNodeNames := map[string]bool{
		"index": true, "app": true, "main": true, "server": true, "start": true, "run": true,
		"node": true, "npm": true, "yarn": true, "nodemon": true, "pm2": true, "forever": true,
	}

	if genericNodeNames[strings.ToLower(baseName)] {
		return ""
	}

	return cleanName(baseName)
}

// extractNodeAgentPathFromCmdline extracts the path following an agent flag
// (e.g. --require, -r) in the command line.
func extractNodeAgentPathFromCmdline(cmdline, pattern string) string {
	idx := strings.Index(strings.ToLower(cmdline), pattern)
	if idx == -1 {
		return ""
	}

	agentPart := cmdline[idx+len(pattern):]
	if spaceIdx := strings.Index(agentPart, " "); spaceIdx != -1 {
		agentPart = agentPart[:spaceIdx]
	}
	return strings.TrimSpace(agentPart)
}

// extractNodeAgentFromEnvValue extracts an agent path from NODE_OPTIONS
// by looking for --require, -r, --import, or --loader flags.
func extractNodeAgentFromEnvValue(envValue string) string {
	agentPatterns := []string{"--require ", "-r ", "--import ", "--loader "}
	for _, pattern := range agentPatterns {
		if idx := strings.Index(envValue, pattern); idx != -1 {
			agentPart := envValue[idx+len(pattern):]
			if spaceIdx := strings.Index(agentPart, " "); spaceIdx != -1 {
				agentPart = agentPart[:spaceIdx]
			}
			return agentPart
		}
	}
	return ""
}
