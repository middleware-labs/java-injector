package main

import (
	// "bufio"
	// "context"
	// "fmt"
	// "io"
	"os"
	// "os/exec"
	// "path/filepath"
	// "regexp"
	// "strings"

	// "github.com/k0kubun/pp"
	// "github.com/middleware-labs/java-injector/pkg/discovery"

	"github.com/middleware-labs/java-injector/pkg/cli"
	// "github.com/middleware-labs/java-injector/pkg/config"
	// "github.com/middleware-labs/java-injector/pkg/docker"
)

const (
	DefaultAgentDir  = "/opt/middleware/agents"
	DefaultAgentName = "middleware-javaagent-1.8.1.jar"
	DefaultAgentPath = DefaultAgentDir + "/" + DefaultAgentName
)

func main() {
	router := cli.NewRouter()
	if err := router.Run(os.Args); err != nil {
		// Error is already printed by the router or command
		os.Exit(1)
	}
}

// if len(os.Args) < 2 {
// 	printUsage()
// 	os.Exit(1)
// }

// command := os.Args[1]

// switch command {
// case "list":
// 	listProcesses()
// case "auto-instrument":
// 	autoInstrument()
// case "uninstrument":
// 	uninstrument()
// case "instrument-docker":
// 	instrumentDocker()
// case "list-docker":
// 	listDockerContainers()
// case "instrument-container":
// 	if len(os.Args) < 3 {
// 		fmt.Println("❌ Container name required")
// 		fmt.Println("Usage: mw-injector instrument-container <container-name>")
// 		os.Exit(1)
// 	}
// 	instrumentSpecificContainer(os.Args[2])
// case "uninstrument-docker":
// 	uninstrumentDocker()
// case "uninstrument-container":
// 	if len(os.Args) < 3 {
// 		fmt.Println("❌ Container name required")
// 		fmt.Println("Usage: mw-injector uninstrument-container <container-name>")
// 		os.Exit(1)
// 	}
// 	uninstrumentSpecificContainer(os.Args[2])
// default:
// 	fmt.Printf("Unknown command: %s\n", command)
// 	printUsage()
// 	os.Exit(1)
// }

// func printUsage() {
// 	fmt.Println(`MW Injector Manager
// Usage:
//   mw-injector list                          List all Java processes (host)
//   mw-injector list-docker                   List all Java Docker containers
//   mw-injector list-all                      List both host processes and Docker containers
//   mw-injector auto-instrument               Auto-instrument all uninstrumented processes (host)
//   mw-injector instrument-docker             Auto-instrument all Java Docker containers
//   mw-injector instrument-container <name>   Instrument specific Docker container
//   mw-injector uninstrument                  Uninstrument all host processes
//   mw-injector uninstrument-docker           Uninstrument all Docker containers
//   mw-injector uninstrument-container <name> Uninstrument specific Docker container

// Examples:
//   # Host Java processes
//   sudo mw-injector list
//   sudo mw-injector auto-instrument
//
//   # Docker containers
//   sudo mw-injector list-docker
//   sudo mw-injector instrument-docker
//   sudo mw-injector instrument-container my-java-app
//   sudo mw-injector uninstrument-container my-java-app
//
//   # List everything
//   sudo mw-injector list-all`)
// }

// func listProcesses() {
// 	ctx := context.Background()

// 	processes, err := discovery.FindAllJavaProcesses(ctx)
// 	if err != nil {
// 		fmt.Printf("Error: %v\n", err)
// 		os.Exit(1)
// 	}

// 	if len(processes) == 0 {
// 		fmt.Println("No Java processes found")
// 		return
// 	}

// 	fmt.Printf("Found %d Java processes:\n\n", len(processes))

// 	for _, proc := range processes {
// 		fmt.Printf("PID: %d\n", proc.ProcessPID)
// 		fmt.Printf("  Service: %s\n", proc.ServiceName)
// 		fmt.Printf("  Owner: %s\n", proc.ProcessOwner)
// 		fmt.Printf("  Agent: %s\n", proc.FormatAgentStatus())

// 		if proc.HasJavaAgent {
// 			agentInfo := proc.GetAgentInfo()
// 			fmt.Printf("  Agent Path: %s\n", agentInfo.Path)
// 		}

// 		// Check if Tomcat
// 		if proc.IsTomcat() {
// 			tomcatInfo := proc.ExtractTomcatInfo()
// 			fmt.Printf("  Type: Tomcat\n")
// 			fmt.Printf("  Instance: %s\n", tomcatInfo.InstanceName)
// 			if len(tomcatInfo.Webapps) > 0 {
// 				fmt.Printf("  Webapps: %v\n", tomcatInfo.Webapps)
// 			}
// 		}

// 		// Check if configured
// 		configPath := getConfigPath(&proc)
// 		if fileExists(configPath) {
// 			fmt.Printf("  Config: ✅ %s\n", configPath)
// 		} else {
// 			fmt.Printf("  Config: ❌ Not configured\n")
// 		}

// 		fmt.Println()
// 	}
// }

// func autoInstrument() {
// 	ctx := context.Background()

// 	// Check if running as root
// 	if os.Geteuid() != 0 {
// 		fmt.Println("❌ This command requires root privileges")
// 		fmt.Println("   Run with: sudo mw-injector auto-instrument")
// 		os.Exit(1)
// 	}

// 	// Get API key and target
// 	reader := bufio.NewReader(os.Stdin)

// 	fmt.Print("Middleware.io API Key: ")
// 	apiKey, _ := reader.ReadString('\n')
// 	apiKey = strings.TrimSpace(apiKey)

// 	if apiKey == "" {
// 		fmt.Println("❌ API key is required")
// 		os.Exit(1)
// 	}

// 	fmt.Print("Target endpoint [https://prod.middleware.io:443]: ")
// 	target, _ := reader.ReadString('\n')
// 	target = strings.TrimSpace(target)
// 	if target == "" {
// 		target = "https://prod.middleware.io:443"
// 	}

// 	fmt.Printf("Java agent path %s: \n", DefaultAgentPath)
// 	agentPath, _ := reader.ReadString('\n')
// 	agentPath = strings.TrimSpace(agentPath)
// 	if agentPath == "" {
// 		agentPath = DefaultAgentPath
// 	}

// 	// Ensure agent is installed and accessible
// 	installedPath, err := EnsureAgentInstalled(agentPath)
// 	if err != nil {
// 		fmt.Printf("failed to prepare agent: %w \n", err)
// 		return
// 	}

// 	// Discover processes
// 	processes, err := discovery.FindAllJavaProcesses(ctx)
// 	if err != nil {
// 		fmt.Printf("❌ Error discovering processes: %v\n", err)
// 		os.Exit(1)
// 	}

// 	if len(processes) == 0 {
// 		fmt.Println("No Java processes found")
// 		return
// 	}

// 	fmt.Printf("\n🔍 Found %d Java processes\n\n", len(processes))

// 	fmt.Printf("\n✅ Using agent at: %s\n", installedPath)
// 	fmt.Printf("   Permissions: world-readable (0644)\n")
// 	fmt.Printf("   Owner: root:root\n")
// 	fmt.Printf("   Accessible by: ALL users\n\n")
// 	configured := 0
// 	updated := 0
// 	skipped := 0
// 	var servicesToRestart []string

// 	for _, proc := range processes {
// 		// 1. DETECT: Use the reliable systemd-run check for this specific process.
// 		if !isAgentAccessibleBySystemd(agentPath, proc.ProcessOwner) {
// 			// 2. INFORM: Tell the user about the specific failure.
// 			fmt.Printf("❌ Skipping PID %d (%s) due to a permission issue.\n", proc.ProcessPID, proc.ServiceName)
// 			fmt.Printf("   └── Reason: The service user '%s' cannot access the agent file within the systemd security context.\n", proc.ProcessOwner)
// 			fmt.Printf("   └── To fix, check file permissions and SELinux/AppArmor policies.\n\n")

// 			// 3. SKIP: Increment the counter and move to the next process.
// 			skipped++
// 			continue
// 		}

// 		// --- If the check passes, the original logic for instrumenting the service continues ---

// 		configPath := getConfigPath(&proc)
// 		shouldUpdate := false

// 		// Check if already configured
// 		if fileExists(configPath) {
// 			fmt.Printf("⚠️  PID %d (%s) is already configured\n", proc.ProcessPID, proc.ServiceName)
// 			fmt.Print("   Update configuration? [y/N]: ")

// 			response, _ := reader.ReadString('\n')
// 			response = strings.TrimSpace(strings.ToLower(response))

// 			if response != "y" && response != "yes" {
// 				fmt.Printf("⏭️  Skipping PID %d (%s)\n\n", proc.ProcessPID, proc.ServiceName)
// 				skipped++
// 				continue
// 			}
// 			shouldUpdate = true
// 		}

// 		// Generate service name and config
// 		var systemdServiceName string
// 		if proc.IsTomcat() {

// 			tomcatInfo := proc.ExtractTomcatInfo()
// 			pp.Println("tomcat-info: ", tomcatInfo)

// 			// Generate service name: tomcat-{instance}
// 			serviceName := generateServiceName(&proc)

// 			// TODO: When MW agent supports pattern expansion via MW_SERVICE_NAME_PATTERN:
// 			// 1. Uncomment the pattern line below
// 			// 2. Update createTomcatConfig to write MW_SERVICE_NAME_PATTERN instead of MW_SERVICE_NAME
// 			// 3. Agent will expand {context} dynamically per webapp
// 			// pattern := fmt.Sprintf("%s-{context}", serviceName)

// 			// For now: Use serviceName as-is (no pattern)
// 			err := createTomcatConfig(configPath, serviceName, serviceName, apiKey, target, agentPath)
// 			if err != nil {
// 				fmt.Printf("❌ Failed to configure PID %d: %v\n", proc.ProcessPID, err)
// 				continue
// 			}

// 			systemdServiceName = "tomcat.service"
// 			err = createSystemdDropIn(systemdServiceName, configPath, true)
// 			if err != nil {
// 				fmt.Printf("❌ Failed to create systemd drop-in for PID %d: %v\n", proc.ProcessPID, err)
// 				continue
// 			}

// 			if shouldUpdate {
// 				fmt.Printf("🔄 Updated Tomcat: %s\n", serviceName)
// 				updated++
// 			} else {
// 				fmt.Printf("✅ Configured Tomcat: %s\n", serviceName)
// 				configured++
// 			}
// 		} else {
// 			serviceName := generateServiceName(&proc)
// 			systemdServiceName = getSystemdServiceName(&proc)

// 			err := createStandardConfig(configPath, serviceName, apiKey, target, agentPath)
// 			if err != nil {
// 				fmt.Printf("❌ Failed to configure PID %d: %v\n", proc.ProcessPID, err)
// 				continue
// 			}

// 			err = createSystemdDropIn(systemdServiceName, configPath, false)
// 			if err != nil {
// 				fmt.Printf("❌ Failed to create systemd drop-in for PID %d: %v\n", proc.ProcessPID, err)
// 				continue
// 			}

// 			if shouldUpdate {
// 				fmt.Printf("🔄 Updated: %s (service: %s)\n", serviceName, systemdServiceName)
// 				updated++
// 			} else {
// 				fmt.Printf("✅ Configured: %s (service: %s)\n", serviceName, systemdServiceName)
// 				configured++
// 			}
// 		}

// 		// Add to restart list if not already there
// 		found := false
// 		for _, s := range servicesToRestart {
// 			if s == systemdServiceName {
// 				found = true
// 				break
// 			}
// 		}
// 		if !found && systemdServiceName != "" {
// 			servicesToRestart = append(servicesToRestart, systemdServiceName)
// 		}
// 		fmt.Println()
// 	}

// 	fmt.Printf("\n🎉 Auto-instrumentation complete!\n")
// 	fmt.Printf("   Configured: %d\n", configured)
// 	fmt.Printf("   Updated:    %d\n", updated)
// 	fmt.Printf("   Skipped:    %d\n", skipped)
// 	fmt.Printf("   Total:      %d\n", len(processes))

// 	// Restart services
// 	if len(servicesToRestart) > 0 {
// 		fmt.Printf("\n🔄 Restarting %d service(s)...\n\n", len(servicesToRestart))

// 		exec.Command("systemctl", "daemon-reload").Run()

// 		for _, service := range servicesToRestart {
// 			fmt.Printf("   Restarting %s...", service)
// 			cmd := exec.Command("systemctl", "restart", service)
// 			err := cmd.Run()

// 			if err != nil {
// 				fmt.Printf(" ❌ Failed\n")
// 				fmt.Printf("       Error: %v\n", err)
// 				fmt.Printf("       Try manually: sudo systemctl restart %s\n", service)
// 			} else {
// 				fmt.Printf(" ✅ Done\n")
// 			}
// 		}
// 		fmt.Println("\n✅ All services restarted!")
// 	}
// }

// func listDockerContainers() {
// 	ctx := context.Background()
// 	discoverer := discovery.NewDockerDiscoverer(ctx)

// 	containers, err := discoverer.DiscoverJavaContainers()
// 	if err != nil {
// 		fmt.Printf("Error: %v\n", err)
// 		os.Exit(1)
// 	}

// 	if len(containers) == 0 {
// 		fmt.Println("No Java Docker containers found")
// 		return
// 	}

// 	fmt.Printf("Found %d Java Docker containers:\n\n", len(containers))

// 	for _, container := range containers {
// 		fmt.Printf("Container: %s\n", container.ContainerName)
// 		fmt.Printf("  ID: %s\n", container.ContainerID[:12])
// 		fmt.Printf("  Image: %s:%s\n", container.ImageName, container.ImageTag)
// 		fmt.Printf("  Status: %s\n", container.Status)
// 		fmt.Printf("  Agent: %s\n", container.FormatAgentStatus())

// 		if container.HasJavaAgent {
// 			fmt.Printf("  Agent Path: %s\n", container.JavaAgentPath)
// 		}

// 		if container.IsCompose {
// 			fmt.Printf("  Type: Docker Compose\n")
// 			fmt.Printf("  Project: %s\n", container.ComposeProject)
// 			fmt.Printf("  Service: %s\n", container.ComposeService)
// 		}

// 		if len(container.JarFiles) > 0 {
// 			fmt.Printf("  JAR Files: %v\n", container.JarFiles)
// 		}

// 		if container.Instrumented {
// 			fmt.Printf("  Status: ✅ Instrumented\n")
// 		} else {
// 			fmt.Printf("  Status: ⚠️  Not instrumented\n")
// 		}

// 		fmt.Println()
// 	}
// }

// func listAll() {
// 	fmt.Println("=" + strings.Repeat("=", 70))
// 	fmt.Println("HOST JAVA PROCESSES")
// 	fmt.Println("=" + strings.Repeat("=", 70))
// 	listProcesses()

// 	fmt.Println("\n" + strings.Repeat("=", 70))
// 	fmt.Println("DOCKER JAVA CONTAINERS")
// 	fmt.Println(strings.Repeat("=", 70))
// 	listDockerContainers()
// }

// func instrumentDocker() {
// 	ctx := context.Background()

// 	// Check if running as root
// 	if os.Geteuid() != 0 {
// 		fmt.Println("❌ This command requires root privileges")
// 		fmt.Println("   Run with: sudo mw-injector instrument-docker")
// 		os.Exit(1)
// 	}

// 	// Get API key and target
// 	reader := bufio.NewReader(os.Stdin)

// 	fmt.Print("Middleware.io API Key: ")
// 	apiKey, _ := reader.ReadString('\n')
// 	apiKey = strings.TrimSpace(apiKey)

// 	if apiKey == "" {
// 		fmt.Println("❌ API key is required")
// 		os.Exit(1)
// 	}

// 	fmt.Print("Target endpoint [https://prod.middleware.io:443]: ")
// 	target, _ := reader.ReadString('\n')
// 	target = strings.TrimSpace(target)
// 	if target == "" {
// 		target = "https://prod.middleware.io:443"
// 	}

// 	fmt.Printf("Java agent path [%s]: ", DefaultAgentPath)
// 	agentPath, _ := reader.ReadString('\n')
// 	agentPath = strings.TrimSpace(agentPath)
// 	if agentPath == "" {
// 		agentPath = DefaultAgentPath
// 	}

// 	// Ensure agent is installed and accessible
// 	installedPath, err := EnsureAgentInstalled(agentPath)
// 	if err != nil {
// 		fmt.Printf("❌ Failed to prepare agent: %v\n", err)
// 		return
// 	}

// 	// Discover Docker containers
// 	discoverer := discovery.NewDockerDiscoverer(ctx)
// 	containers, err := discoverer.DiscoverJavaContainers()
// 	if err != nil {
// 		fmt.Printf("❌ Error discovering containers: %v\n", err)
// 		os.Exit(1)
// 	}

// 	if len(containers) == 0 {
// 		fmt.Println("No Java Docker containers found")
// 		return
// 	}

// 	fmt.Printf("\n🔍 Found %d Java Docker containers\n\n", len(containers))
// 	fmt.Printf("✅ Using agent at: %s\n\n", installedPath)

// 	configured := 0
// 	updated := 0
// 	skipped := 0

// 	dockerOps := docker.NewDockerOperations(ctx, installedPath)

// 	for _, container := range containers {
// 		// Skip if already instrumented
// 		if container.Instrumented && container.IsMiddlewareAgent {
// 			fmt.Printf("✅ Container %s is already instrumented\n", container.ContainerName)
// 			fmt.Print("   Update configuration? [y/N]: ")

// 			response, _ := reader.ReadString('\n')
// 			response = strings.TrimSpace(strings.ToLower(response))

// 			if response != "y" && response != "yes" {
// 				fmt.Printf("⏭️  Skipping container %s\n\n", container.ContainerName)
// 				skipped++
// 				continue
// 			}
// 		}

// 		// Create configuration
// 		cfg := config.DefaultConfiguration()
// 		cfg.MWAPIKey = apiKey
// 		cfg.MWTarget = target
// 		cfg.MWServiceName = container.GetServiceName()
// 		cfg.JavaAgentPath = docker.DefaultContainerAgentPath

// 		// Instrument container
// 		err := dockerOps.InstrumentContainer(container.ContainerName, &cfg)
// 		if err != nil {
// 			fmt.Printf("❌ Failed to instrument container %s: %v\n", container.ContainerName, err)
// 			skipped++
// 		} else {
// 			if container.Instrumented {
// 				updated++
// 			} else {
// 				configured++
// 			}
// 		}
// 		fmt.Println()
// 	}

// 	fmt.Printf("\n🎉 Docker instrumentation complete!\n")
// 	fmt.Printf("   Configured: %d\n", configured)
// 	fmt.Printf("   Updated: %d\n", updated)
// 	fmt.Printf("   Skipped: %d\n", skipped)

// 	if configured > 0 || updated > 0 {
// 		fmt.Println("\n📊 Containers are now sending telemetry data to Middleware.io")
// 	}
// }

// func instrumentSpecificContainer(containerName string) {
// 	ctx := context.Background()

// 	// Check if running as root
// 	if os.Geteuid() != 0 {
// 		fmt.Println("❌ This command requires root privileges")
// 		fmt.Println("   Run with: sudo mw-injector instrument-container " + containerName)
// 		os.Exit(1)
// 	}

// 	// Get API key and target
// 	reader := bufio.NewReader(os.Stdin)

// 	fmt.Print("Middleware.io API Key: ")
// 	apiKey, _ := reader.ReadString('\n')
// 	apiKey = strings.TrimSpace(apiKey)

// 	if apiKey == "" {
// 		fmt.Println("❌ API key is required")
// 		os.Exit(1)
// 	}

// 	fmt.Print("Target endpoint [https://prod.middleware.io:443]: ")
// 	target, _ := reader.ReadString('\n')
// 	target = strings.TrimSpace(target)
// 	if target == "" {
// 		target = "https://prod.middleware.io:443"
// 	}

// 	fmt.Printf("Java agent path [%s]: ", DefaultAgentPath)
// 	agentPath, _ := reader.ReadString('\n')
// 	agentPath = strings.TrimSpace(agentPath)
// 	if agentPath == "" {
// 		agentPath = DefaultAgentPath
// 	}

// 	// Ensure agent is installed
// 	installedPath, err := EnsureAgentInstalled(agentPath)
// 	if err != nil {
// 		fmt.Printf("❌ Failed to prepare agent: %v\n", err)
// 		return
// 	}

// 	// Verify container exists
// 	discoverer := discovery.NewDockerDiscoverer(ctx)
// 	container, err := discoverer.GetContainerByName(containerName)
// 	if err != nil {
// 		fmt.Printf("❌ Container not found: %v\n", err)
// 		os.Exit(1)
// 	}

// 	fmt.Printf("\n🔍 Found container: %s\n", container.ContainerName)
// 	fmt.Printf("   Image: %s:%s\n", container.ImageName, container.ImageTag)
// 	fmt.Printf("   Status: %s\n\n", container.Status)

// 	// Create configuration
// 	cfg := config.DefaultConfiguration()
// 	cfg.MWAPIKey = apiKey
// 	cfg.MWTarget = target
// 	cfg.MWServiceName = container.GetServiceName()
// 	cfg.JavaAgentPath = docker.DefaultContainerAgentPath

// 	// Instrument
// 	dockerOps := docker.NewDockerOperations(ctx, installedPath)
// 	if err := dockerOps.InstrumentContainer(containerName, &cfg); err != nil {
// 		fmt.Printf("❌ Failed to instrument container: %v\n", err)
// 		os.Exit(1)
// 	}

// 	fmt.Println("\n🎉 Container instrumented successfully!")
// 	fmt.Println("📊 Container is now sending telemetry data to Middleware.io")
// }

// func uninstrumentDocker() {
// 	ctx := context.Background()

// 	// Check if running as root
// 	if os.Geteuid() != 0 {
// 		fmt.Println("❌ This command requires root privileges")
// 		fmt.Println("   Run with: sudo mw-injector uninstrument-docker")
// 		os.Exit(1)
// 	}

// 	reader := bufio.NewReader(os.Stdin)

// 	fmt.Print("Uninstrument ALL Docker containers? [y/N]: ")
// 	response, _ := reader.ReadString('\n')
// 	response = strings.TrimSpace(strings.ToLower(response))

// 	if response != "y" && response != "yes" {
// 		fmt.Println("Cancelled")
// 		return
// 	}

// 	dockerOps := docker.NewDockerOperations(ctx, DefaultAgentPath)

// 	// List instrumented containers
// 	instrumented, err := dockerOps.ListInstrumentedContainers()
// 	if err != nil {
// 		fmt.Printf("❌ Error listing instrumented containers: %v\n", err)
// 		os.Exit(1)
// 	}

// 	if len(instrumented) == 0 {
// 		fmt.Println("No instrumented Docker containers found")
// 		return
// 	}

// 	fmt.Printf("\n🔧 Uninstrumenting %d containers...\n\n", len(instrumented))

// 	success := 0
// 	failed := 0

// 	for _, container := range instrumented {
// 		err := dockerOps.UninstrumentContainer(container.ContainerName)
// 		if err != nil {
// 			fmt.Printf("❌ Failed to uninstrument %s: %v\n", container.ContainerName, err)
// 			failed++
// 		} else {
// 			success++
// 		}
// 	}

// 	fmt.Printf("\n🎉 Uninstrumentation complete!\n")
// 	fmt.Printf("   Success: %d\n", success)
// 	fmt.Printf("   Failed: %d\n", failed)
// }

// func uninstrumentSpecificContainer(containerName string) {
// 	ctx := context.Background()

// 	// Check if running as root
// 	if os.Geteuid() != 0 {
// 		fmt.Println("❌ This command requires root privileges")
// 		fmt.Println("   Run with: sudo mw-injector uninstrument-container " + containerName)
// 		os.Exit(1)
// 	}

// 	dockerOps := docker.NewDockerOperations(ctx, DefaultAgentPath)

// 	fmt.Printf("🔧 Uninstrumenting container: %s\n\n", containerName)

// 	if err := dockerOps.UninstrumentContainer(containerName); err != nil {
// 		fmt.Printf("❌ Failed to uninstrument container: %v\n", err)
// 		os.Exit(1)
// 	}

// 	fmt.Println("🎉 Container uninstrumented successfully!")
// }

// // Keep all existing functions from the original main.go
// // (listProcesses, autoInstrument, uninstrument, etc.)

// func createSystemdDropIn(serviceName string, configPath string, isTomcat bool) error {
// 	// Read the config file to get actual values
// 	configVars, err := readConfigFile(configPath)
// 	if err != nil {
// 		return fmt.Errorf("failed to read config file: %v", err)
// 	}
// 	pp.Println(configVars)

// 	hostname, err := os.Hostname()
// 	if err != nil {
// 		hostname = "unknown"
// 	}
// 	var dropInContent string

// 	if isTomcat {
// 		// 		dropInContent LINA_OPTS from main service file
// 		existingOpts := readExistingCatalinaOpts(serviceName)

// 		// Build full CATALINA_OPTS with agent appended
// 		fullOpts := fmt.Sprintf("%s -javaagent:%s", existingOpts, configVars["MW_JAVA_AGENT_PATH"])

// 		serviceNameWithHost := fmt.Sprintf("%s@%s", configVars["MW_SERVICE_NAME_PATTERN"], hostname)

// 		dropInContent = fmt.Sprintf(`[Service]
// # Grant read access to Middleware agent
// ReadOnlyPaths=%s

// # Tomcat options with Middleware agent (hardcoded - systemd doesn't support variable expansion)
// Environment="CATALINA_OPTS=%s"

// # OpenTelemetry configuration
// Environment="OTEL_SERVICE_NAME=%s"
// Environment="OTEL_EXPORTER_OTLP_ENDPOINT=%s"
// Environment="OTEL_EXPORTER_OTLP_HEADERS=authorization=%s"
// Environment="OTEL_TRACES_EXPORTER=otlp"
// Environment="OTEL_METRICS_EXPORTER=otlp"
// Environment="OTEL_LOGS_EXPORTER=otlp"
// `,
// 			configVars["MW_JAVA_AGENT_PATH"],
// 			fullOpts,
// 			serviceNameWithHost,
// 			configVars["MW_TARGET"],
// 			configVars["MW_API_KEY"])
// 	} else {
// 		dropInContent = fmt.Sprintf(`[Service]
// Environment="JAVA_TOOL_OPTIONS=-javaagent:%s"
// Environment="OTEL_SERVICE_NAME=%s"
// Environment="OTEL_EXPORTER_OTLP_ENDPOINT=%s"
// Environment="OTEL_EXPORTER_OTLP_HEADERS=authorization=%s"
// Environment="OTEL_TRACES_EXPORTER=otlp"
// Environment="OTEL_METRICS_EXPORTER=otlp"
// Environment="OTEL_LOGS_EXPORTER=otlp"
// `, configVars["MW_JAVA_AGENT_PATH"], configVars["MW_SERVICE_NAME"],
// 			configVars["MW_TARGET"], configVars["MW_API_KEY"])
// 	}

// 	// Create drop-in directory
// 	dropInDir := fmt.Sprintf("/etc/systemd/system/%s.d", serviceName)
// 	if err := os.MkdirAll(dropInDir, 0o755); err != nil {
// 		return fmt.Errorf("failed to create drop-in directory: %v", err)
// 	}

// 	// Write drop-in file
// 	dropInPath := filepath.Join(dropInDir, "middleware-instrumentation.conf")
// 	if err := os.WriteFile(dropInPath, []byte(dropInContent), 0o644); err != nil {
// 		return fmt.Errorf("failed to write drop-in file: %v", err)
// 	}

// 	fmt.Printf("   Created drop-in: %s\n", dropInPath)
// 	return nil
// }

// // readExistingCatalinaOpts reads CATALINA_OPTS from the main service file
// func readExistingCatalinaOpts(serviceName string) string {
// 	// Try multiple possible locations
// 	possiblePaths := []string{
// 		fmt.Sprintf("/etc/systemd/system/%s.service", serviceName),
// 		fmt.Sprintf("/lib/systemd/system/%s.service", serviceName),
// 		fmt.Sprintf("/usr/lib/systemd/system/%s.service", serviceName),
// 	}

// 	for _, path := range possiblePaths {
// 		if opts := extractCatalinaOpts(path); opts != "" {
// 			return opts
// 		}
// 	}

// 	// Default if not found
// 	return "-Xms512M -Xmx1024M -server -XX:+UseParallelGC"
// }

// // extractCatalinaOpts extracts CATALINA_OPTS value from a service file
// func extractCatalinaOpts(serviceFile string) string {
// 	file, err := os.Open(serviceFile)
// 	if err != nil {
// 		return ""
// 	}
// 	defer file.Close()

// 	scanner := bufio.NewScanner(file)
// 	for scanner.Scan() {
// 		line := strings.TrimSpace(scanner.Text())

// 		// Look for: Environment="CATALINA_OPTS=..."
// 		if strings.HasPrefix(line, "Environment=\"CATALINA_OPTS=") {
// 			// Extract the value between quotes
// 			start := strings.Index(line, "CATALINA_OPTS=") + len("CATALINA_OPTS=")
// 			end := strings.LastIndex(line, "\"")
// 			if start > 0 && end > start {
// 				return line[start:end]
// 			}
// 		}
// 	}

// 	return ""
// }

// func readConfigFile(path string) (map[string]string, error) {
// 	file, err := os.Open(path)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer file.Close()

// 	vars := make(map[string]string)
// 	scanner := bufio.NewScanner(file)

// 	for scanner.Scan() {
// 		line := strings.TrimSpace(scanner.Text())

// 		// Skip comments and empty lines
// 		if line == "" || strings.HasPrefix(line, "#") {
// 			continue
// 		}

// 		// Parse KEY=VALUE
// 		parts := strings.SplitN(line, "=", 2)
// 		if len(parts) == 2 {
// 			key := strings.TrimSpace(parts[0])
// 			value := strings.TrimSpace(parts[1])
// 			vars[key] = value
// 		}
// 	}

// 	if err := scanner.Err(); err != nil {
// 		return nil, err
// 	}

// 	return vars, nil
// }

// func getConfigPath(proc *discovery.JavaProcess) string {
// 	serviceName := generateServiceName(proc)

// 	if proc.IsTomcat() {
// 		return fmt.Sprintf("/etc/middleware/tomcat/%s.conf", serviceName)
// 	}

// 	deploymentType := detectDeploymentType(proc)
// 	return fmt.Sprintf("/etc/middleware/%s/%s.conf", deploymentType, serviceName)
// }

// func getSystemdServiceName(proc *discovery.JavaProcess) string {
// 	// Try to find the actual systemd service by PID
// 	cmd := exec.Command("systemctl", "status", fmt.Sprintf("%d", proc.ProcessPID))
// 	output, err := cmd.CombinedOutput()

// 	if err == nil {
// 		// Parse output to find service name
// 		lines := strings.Split(string(output), "\n")
// 		if len(lines) > 0 {
// 			firstLine := lines[0]
// 			if strings.HasPrefix(firstLine, "●") || strings.HasPrefix(firstLine, "○") {
// 				parts := strings.Fields(firstLine)
// 				if len(parts) >= 2 {
// 					return parts[1]
// 				}
// 			}
// 		}
// 	}

// 	// Fallback: try common service name patterns
// 	serviceName := generateServiceName(proc)
// 	possibleNames := []string{
// 		"spring-book.service",
// 		serviceName + ".service",
// 		proc.ServiceName + ".service",
// 	}

// 	for _, name := range possibleNames {
// 		cmd := exec.Command("systemctl", "status", name)
// 		if err := cmd.Run(); err == nil {
// 			return name
// 		}
// 	}

// 	return serviceName + ".service"
// }

// func detectDeploymentType(proc *discovery.JavaProcess) string {
// 	if proc.ProcessOwner != "root" && proc.ProcessOwner != os.Getenv("USER") {
// 		return "systemd"
// 	}
// 	return "standalone"
// }

// func generateServiceName(proc *discovery.JavaProcess) string {
// 	// For Tomcat services, use tomcat-{INSTANCE-NAME} pattern
// 	if proc.IsTomcat() {
// 		tomcatInfo := proc.ExtractTomcatInfo()

// 		// Get instance name from CATALINA_BASE
// 		instanceName := filepath.Base(filepath.Dir(tomcatInfo.CatalinaBase))

// 		// Fallback: try CATALINA_BASE itself
// 		if instanceName == "." || instanceName == "/" || instanceName == "opt" {
// 			instanceName = filepath.Base(tomcatInfo.CatalinaBase)
// 		}

// 		// Clean the instance name (removes version numbers, apache- prefix, etc.)
// 		instanceName = cleanTomcatInstance(instanceName)

// 		// Handle edge cases
// 		if instanceName == "" || instanceName == "tomcat" {
// 			instanceName = "default"
// 		}

// 		// TODO: Once MW agent supports MW_SERVICE_NAME_PATTERN with {context} expansion,
// 		// we can return a pattern here instead of just the instance name.
// 		// For now, just return: tomcat-{instance}
// 		// Future: return pattern that agent will expand per-webapp
// 		return fmt.Sprintf("tomcat-%s", instanceName)
// 	}

// 	// For non-Tomcat services, use JAR name as default
// 	if proc.JarFile != "" {
// 		return cleanJarName(proc.JarFile)
// 	}

// 	// Last resort fallback
// 	if proc.ServiceName != "" && proc.ServiceName != "java-service" {
// 		return cleanServiceName(proc.ServiceName)
// 	}

// 	return fmt.Sprintf("java-app-%d", proc.ProcessPID)
// }

// func cleanTomcatInstance(name string) string {
// 	pp.Println("cleaningTomcat Instance. Name: ", name)
// 	name = filepath.Base(name)
// 	name = strings.TrimPrefix(name, "apache-")
// 	re := regexp.MustCompile(`-\d+\.\d+\.\d+.*$`)
// 	name = re.ReplaceAllString(name, "")

// 	return cleanServiceName(name)
// }

// func cleanJarName(jar string) string {
// 	name := strings.TrimSuffix(jar, ".jar")
// 	patterns := []string{
// 		`-\d+\.\d+\.\d+.*$`,
// 		`-SNAPSHOT$`,
// 		`_\d+\.\d+\.\d+.*$`,
// 		`-BUILD-\d+$`,
// 	}

// 	for _, pattern := range patterns {
// 		re := regexp.MustCompile(pattern)
// 		name = re.ReplaceAllString(name, "")
// 	}
// 	return cleanServiceName(name)
// }

// func cleanWebappName(name string) string {
// 	return cleanJarName(name)
// }

// func cleanServiceName(name string) string {
// 	name = strings.ToLower(name)
// 	name = strings.ReplaceAll(name, "_", "-")
// 	re := regexp.MustCompile(`[^a-z0-9\-]+`)
// 	name = re.ReplaceAllString(name, "")
// 	name = strings.Trim(name, "-")
// 	re = regexp.MustCompile(`-+`)
// 	name = re.ReplaceAllString(name, "-")
// 	pp.Println("Clean service Name: ", name)
// 	return name
// }

// func createTomcatConfig(configPath, instanceName, pattern, apiKey, target, agentPath string) error {
// 	dir := filepath.Dir(configPath)
// 	if err := os.MkdirAll(dir, 0o755); err != nil {
// 		return err
// 	}

// 	content := fmt.Sprintf(`# Middleware.io Configuration for Tomcat
// # Instance: %s
// # Generated: %s

// # Dynamic service naming for webapps
// MW_SERVICE_NAME_PATTERN=%s
// MW_TOMCAT_INSTANCE=%s

// # Middleware.io settings
// MW_API_KEY=%s
// MW_TARGET=%s
// MW_LOG_LEVEL=INFO

// # Java agent
// MW_JAVA_AGENT_PATH=%s

// # Telemetry collection
// MW_APM_COLLECT_TRACES=true
// MW_APM_COLLECT_METRICS=true
// MW_APM_COLLECT_LOGS=true
// `, instanceName, getCurrentTime(), pattern, instanceName, apiKey, target, agentPath)

// 	return os.WriteFile(configPath, []byte(content), 0o644)
// }

// func createStandardConfig(configPath, serviceName, apiKey, target, agentPath string) error {
// 	dir := filepath.Dir(configPath)
// 	if err := os.MkdirAll(dir, 0o755); err != nil {
// 		return err
// 	}

// 	content := fmt.Sprintf(`# Middleware.io Configuration
// # Service: %s
// # Generated: %s

// # Service identification
// MW_SERVICE_NAME=%s

// # Middleware.io settings
// MW_API_KEY=%s
// MW_TARGET=%s
// MW_LOG_LEVEL=INFO

// # Java agent
// MW_JAVA_AGENT_PATH=%s

// # Telemetry collection
// MW_APM_COLLECT_TRACES=true
// MW_APM_COLLECT_METRICS=true
// MW_APM_COLLECT_LOGS=true
// `, serviceName, getCurrentTime(), serviceName, apiKey, target, agentPath)

// 	return os.WriteFile(configPath, []byte(content), 0o644)
// }

// func fileExists(path string) bool {
// 	_, err := os.Stat(path)
// 	return err == nil
// }

// func getCurrentTime() string {
// 	return "2025-10-15 00:00:00"
// }

// func uninstrument() {
// 	ctx := context.Background()

// 	// Check if running as root
// 	if os.Geteuid() != 0 {
// 		fmt.Println("❌ This command requires root privileges")
// 		fmt.Println("   Run with: sudo mw-injector uninstrument")
// 		os.Exit(1)
// 	}

// 	reader := bufio.NewReader(os.Stdin)
// 	// Discover processes
// 	processes, err := discovery.FindAllJavaProcesses(ctx)
// 	if err != nil {
// 		fmt.Printf("❌ Error discovering processes: %v\n", err)
// 		os.Exit(1)
// 	}

// 	if len(processes) == 0 {
// 		fmt.Println("No Running Java processes found")
// 	}

// 	fmt.Printf("\n🔍 Found %d Java processes\n\n", len(processes))

// 	// Check for orphaned configs (services that are stopped/crashed)
// 	orphanedConfigs := findOrphanedConfigs(processes)

// 	if len(processes) == 0 && len(orphanedConfigs) == 0 {
// 		fmt.Println("\nNo instrumented services found")
// 		return
// 	}

// 	if len(orphanedConfigs) > 0 {
// 		fmt.Printf("\n⚠️  Found %d orphaned configuration(s) for stopped/crashed services:\n\n", len(orphanedConfigs))
// 	}

// 	removed := 0
// 	skipped := 0
// 	servicesToRestart := []string{}

// 	// Process orphaned configs first
// 	for _, orphan := range orphanedConfigs {
// 		fmt.Printf("⚠️  Orphaned config found\n")
// 		fmt.Printf("   Service: %s (%s)\n", orphan.ServiceName, orphan.ConfigPath)
// 		if orphan.IsTomcat {
// 			fmt.Printf("   Type: Tomcat (service may be crashed)\n")
// 		} else {
// 			fmt.Printf("   Type: Systemd service (service may be stopped)\n")
// 		}
// 		fmt.Print("   Remove instrumentation? [y/N]: ")

// 		response, _ := reader.ReadString('\n')
// 		response = strings.TrimSpace(strings.ToLower(response))

// 		if response == "y" || response == "yes" {
// 			removeOrphanedConfig(orphan)
// 			removed++

// 			// Add to restart list
// 			if orphan.IsTomcat {
// 				servicesToRestart = append(servicesToRestart, "tomcat.service")
// 			} else {
// 				servicesToRestart = append(servicesToRestart, orphan.ServiceName+".service")
// 			}
// 		} else {
// 			skipped++
// 		}
// 		fmt.Println()
// 	}

// 	// Now process running processes
// 	if len(processes) > 0 {
// 		fmt.Printf("\n📋 Processing running services:\n\n")
// 	}

// 	for _, proc := range processes {
// 		configPath := getConfigPath(&proc)

// 		// Check if configured
// 		if !fileExists(configPath) {
// 			fmt.Printf("⏭️  Skipping PID %d (%s) - not configured\n", proc.ProcessPID, proc.ServiceName)
// 			skipped++
// 			continue
// 		}

// 		fmt.Printf("⚠️  PID %d (%s) is instrumented\n", proc.ProcessPID, proc.ServiceName)
// 		fmt.Print("   Remove instrumentation? [y/N]: ")

// 		response, _ := reader.ReadString('\n')
// 		response = strings.TrimSpace(strings.ToLower(response))

// 		if response != "y" && response != "yes" {
// 			fmt.Printf("⏭️  Skipping PID %d (%s)\n\n", proc.ProcessPID, proc.ServiceName)
// 			skipped++
// 			continue
// 		}

// 		// Remove config file
// 		if err := os.Remove(configPath); err != nil {
// 			fmt.Printf("❌ Failed to remove config for PID %d: %v\n", proc.ProcessPID, err)
// 			continue
// 		}
// 		fmt.Printf("   Removed config: %s\n", configPath)

// 		// Remove systemd drop-in
// 		var systemdServiceName string
// 		if proc.IsTomcat() {
// 			systemdServiceName = "tomcat.service"
// 		} else {
// 			systemdServiceName = getSystemdServiceName(&proc)
// 		}

// 		dropInDir := fmt.Sprintf("/etc/systemd/system/%s.d", systemdServiceName)
// 		dropInPath := filepath.Join(dropInDir, "middleware-instrumentation.conf")

// 		if fileExists(dropInPath) {
// 			if err := os.Remove(dropInPath); err != nil {
// 				fmt.Printf("❌ Failed to remove drop-in for PID %d: %v\n", proc.ProcessPID, err)
// 			} else {
// 				fmt.Printf("   Removed drop-in: %s\n", dropInPath)
// 			}

// 			// Remove directory if empty
// 			files, _ := os.ReadDir(dropInDir)
// 			if len(files) == 0 {
// 				os.Remove(dropInDir)
// 			}
// 		}

// 		if proc.IsTomcat() {
// 			fmt.Printf("🗑️  Removed instrumentation from Tomcat\n")
// 		} else {
// 			serviceName := generateServiceName(&proc)
// 			fmt.Printf("🗑️  Removed instrumentation from: %s\n", serviceName)
// 		}

// 		servicesToRestart = append(servicesToRestart, systemdServiceName)
// 		removed++
// 		fmt.Println()
// 	}

// 	fmt.Printf("\n🎉 Uninstrumentation complete!\n")
// 	fmt.Printf("   Removed: %d\n", removed)
// 	fmt.Printf("   Skipped: %d\n", skipped)
// 	fmt.Printf("   Total: %d\n", len(processes))

// 	// Restart services
// 	if len(servicesToRestart) > 0 {
// 		fmt.Printf("\n🔄 Restarting %d service(s)...\n\n", len(servicesToRestart))

// 		// Reload systemd daemon first
// 		exec.Command("systemctl", "daemon-reload").Run()

// 		for _, service := range servicesToRestart {
// 			fmt.Printf("   Restarting %s...", service)

// 			// Restart the service
// 			cmd := exec.Command("systemctl", "restart", service)
// 			err := cmd.Run()

// 			if err != nil {
// 				fmt.Printf(" ❌ Failed\n")
// 				fmt.Printf("      Error: %v\n", err)
// 				fmt.Printf("      Try manually: sudo systemctl restart %s\n", service)
// 			} else {
// 				fmt.Printf(" ✅ Done\n")
// 			}
// 		}

// 		fmt.Println("\n✅ All services restarted!")
// 	}
// }

// // validateAgentPath checks if the agent path is valid and accessible by the service user
// func validateAgentPath(agentPath string, proc *discovery.JavaProcess) error {
// 	// Check if file exists
// 	if !fileExists(agentPath) {
// 		return fmt.Errorf("agent file does not exist: %s", agentPath)
// 	}

// 	// Check if it's a JAR file
// 	if !strings.HasSuffix(agentPath, ".jar") {
// 		return fmt.Errorf("agent file must be a .jar file: %s", agentPath)
// 	}

// 	// Check if the service user can read the file
// 	username := proc.ProcessOwner

// 	// Test file access for the user
// 	cmd := exec.Command("sudo", "-u", username, "test", "-r", agentPath)
// 	if err := cmd.Run(); err != nil {
// 		return fmt.Errorf("user '%s' cannot read agent file: %s", username, agentPath)
// 	}

// 	return nil
// }

// // ensureAgentAccessible copies the agent to a shared location if needed
// func ensureAgentAccessible(agentPath string, proc *discovery.JavaProcess) (string, error) {
// 	// First check if it's already accessible
// 	if err := validateAgentPath(agentPath, proc); err == nil {
// 		return agentPath, nil
// 	}

// 	// If not accessible, offer to copy to shared location
// 	fmt.Printf("\n⚠️  Warning: User '%s' cannot access %s\n", proc.ProcessOwner, agentPath)
// 	fmt.Print("   Copy agent to /opt/middleware/agents/? [Y/n]: ")

// 	reader := bufio.NewReader(os.Stdin)
// 	response, _ := reader.ReadString('\n')
// 	response = strings.TrimSpace(strings.ToLower(response))

// 	if response == "n" || response == "no" {
// 		return "", fmt.Errorf("agent not accessible and user declined to copy")
// 	}

// 	// Create shared directory
// 	sharedDir := "/opt/middleware/agents"
// 	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
// 		return "", fmt.Errorf("failed to create shared directory: %v", err)
// 	}

// 	// Copy the agent
// 	agentName := filepath.Base(agentPath)
// 	newPath := filepath.Join(sharedDir, agentName)

// 	// Read source file
// 	data, err := os.ReadFile(agentPath)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read agent file: %v", err)
// 	}

// 	// Write to new location
// 	if err := os.WriteFile(newPath, data, 0o644); err != nil {
// 		return "", fmt.Errorf("failed to write agent file: %v", err)
// 	}

// 	// Set appropriate permissions
// 	if err := os.Chmod(newPath, 0o644); err != nil {
// 		return "", fmt.Errorf("failed to set permissions: %v", err)
// 	}

// 	fmt.Printf("   ✅ Copied agent to: %s\n", newPath)

// 	// Verify the new path is accessible
// 	if err := validateAgentPath(newPath, proc); err != nil {
// 		return "", fmt.Errorf("copied agent still not accessible: %v", err)
// 	}

// 	return newPath, nil
// }

// // ensureAgentAccessibleForAll validates agent for all process users at once
// func ensureAgentAccessibleForAll(agentPath string, processes []discovery.JavaProcess) (string, error) {
// 	// Check if file exists
// 	if !fileExists(agentPath) {
// 		return "", fmt.Errorf("agent file does not exist: %s", agentPath)
// 	}

// 	if !strings.HasSuffix(agentPath, ".jar") {
// 		return "", fmt.Errorf("agent file must be a .jar file: %s", agentPath)
// 	}

// 	// Collect all unique users
// 	users := make(map[string]bool)
// 	for _, proc := range processes {
// 		users[proc.ProcessOwner] = true
// 	}

// 	// Check accessibility for each user
// 	inaccessibleUsers := []string{}
// 	for user := range users {
// 		cmd := exec.Command("sudo", "-u", user, "test", "-r", agentPath)
// 		if err := cmd.Run(); err != nil {
// 			inaccessibleUsers = append(inaccessibleUsers, user)
// 		}
// 	}

// 	// If all users can access, we're done
// 	if len(inaccessibleUsers) == 0 {
// 		fmt.Printf("✅ Agent is accessible by all service users\n")
// 		return agentPath, nil
// 	}

// 	// Show warning
// 	fmt.Printf("\n⚠️  Warning: The following users cannot access %s:\n", agentPath)
// 	for _, user := range inaccessibleUsers {
// 		fmt.Printf("   - %s\n", user)
// 	}
// 	fmt.Print("\n   Copy agent to /opt/middleware/agents/ with proper permissions? [Y/n]: ")

// 	reader := bufio.NewReader(os.Stdin)
// 	response, _ := reader.ReadString('\n')
// 	response = strings.TrimSpace(strings.ToLower(response))

// 	if response == "n" || response == "no" {
// 		return "", fmt.Errorf("agent not accessible and user declined to copy")
// 	}

// 	// Copy to shared location
// 	sharedDir := "/opt/middleware/agents"
// 	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
// 		return "", fmt.Errorf("failed to create shared directory: %v", err)
// 	}

// 	agentName := filepath.Base(agentPath)
// 	newPath := filepath.Join(sharedDir, agentName)

// 	data, err := os.ReadFile(agentPath)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read agent file: %v", err)
// 	}

// 	if err := os.WriteFile(newPath, data, 0o644); err != nil {
// 		return "", fmt.Errorf("failed to write agent file: %v", err)
// 	}

// 	fmt.Printf("   ✅ Copied agent to: %s\n", newPath)
// 	fmt.Printf("   Permissions: -rw-r--r-- (readable by all users)\n")

// 	return newPath, nil
// }

// type OrphanedConfig struct {
// 	ConfigPath  string
// 	ServiceName string
// 	IsTomcat    bool
// }

// func findOrphanedConfigs(runningProcesses []discovery.JavaProcess) []OrphanedConfig {
// 	var orphaned []OrphanedConfig

// 	// Get all running process config paths
// 	runningConfigs := make(map[string]bool)
// 	for _, proc := range runningProcesses {
// 		configPath := getConfigPath(&proc)
// 		runningConfigs[configPath] = true
// 	}

// 	// Check systemd configs
// 	systemdDir := "/etc/middleware/systemd"
// 	if fileExists(systemdDir) {
// 		files, _ := os.ReadDir(systemdDir)
// 		for _, file := range files {
// 			if strings.HasSuffix(file.Name(), ".conf") {
// 				configPath := filepath.Join(systemdDir, file.Name())
// 				if !runningConfigs[configPath] {
// 					serviceName := strings.TrimSuffix(file.Name(), ".conf")
// 					orphaned = append(orphaned, OrphanedConfig{
// 						ConfigPath:  configPath,
// 						ServiceName: serviceName,
// 						IsTomcat:    false,
// 					})
// 				}
// 			}
// 		}
// 	}

// 	// Check tomcat configs
// 	tomcatDir := "/etc/middleware/tomcat"
// 	if fileExists(tomcatDir) {
// 		files, _ := os.ReadDir(tomcatDir)
// 		for _, file := range files {
// 			if strings.HasSuffix(file.Name(), ".conf") {
// 				configPath := filepath.Join(tomcatDir, file.Name())
// 				if !runningConfigs[configPath] {
// 					instanceName := strings.TrimSuffix(file.Name(), ".conf")
// 					orphaned = append(orphaned, OrphanedConfig{
// 						ConfigPath:  configPath,
// 						ServiceName: instanceName,
// 						IsTomcat:    true,
// 					})
// 				}
// 			}
// 		}
// 	}

// 	// Check standalone configs
// 	standaloneDir := "/etc/middleware/standalone"
// 	if fileExists(standaloneDir) {
// 		files, _ := os.ReadDir(standaloneDir)
// 		for _, file := range files {
// 			if strings.HasSuffix(file.Name(), ".conf") {
// 				configPath := filepath.Join(standaloneDir, file.Name())
// 				if !runningConfigs[configPath] {
// 					serviceName := strings.TrimSuffix(file.Name(), ".conf")
// 					orphaned = append(orphaned, OrphanedConfig{
// 						ConfigPath:  configPath,
// 						ServiceName: serviceName,
// 						IsTomcat:    false,
// 					})
// 				}
// 			}
// 		}
// 	}

// 	return orphaned
// }

// func removeOrphanedConfig(config OrphanedConfig) {
// 	// Remove config file
// 	if err := os.Remove(config.ConfigPath); err != nil {
// 		fmt.Printf("   ❌ Failed to remove config: %v\n", err)
// 		return
// 	}
// 	fmt.Printf("   Removed config: %s\n", config.ConfigPath)

// 	// Determine systemd service name
// 	var serviceName string
// 	if config.IsTomcat {
// 		serviceName = "tomcat.service"
// 	} else {
// 		// Try to find the actual service name
// 		serviceName = config.ServiceName + ".service"
// 	}

// 	// Remove systemd drop-in
// 	dropInDir := fmt.Sprintf("/etc/systemd/system/%s.d", serviceName)
// 	dropInPath := filepath.Join(dropInDir, "middleware-instrumentation.conf")

// 	if fileExists(dropInPath) {
// 		if err := os.Remove(dropInPath); err == nil {
// 			fmt.Printf("   Removed drop-in: %s\n", dropInPath)
// 		}

// 		// Remove directory if empty
// 		files, _ := os.ReadDir(dropInDir)
// 		if len(files) == 0 {
// 			os.Remove(dropInDir)
// 			fmt.Printf("   Removed empty directory: %s\n", dropInDir)
// 		}
// 	}

// 	fmt.Printf("   🗑️  Removed orphaned instrumentation for: %s\n", config.ServiceName)
// }

// func isAgentAccessibleByUser(agentPath string, username string) bool {
// 	// We use 'sudo -u' to run the command as the target user.
// 	// 'test -r' checks if the file exists and is readable.
// 	// This will fail if the user cannot read the file OR execute any parent directory.
// 	cmd := exec.Command("sudo", "-u", username, "test", "-r", agentPath)

// 	// We only care if the command succeeds (err == nil) or fails.
// 	// A non-zero exit code (err != nil) means access is denied.
// 	err := cmd.Run()
// 	return err == nil
// } // This is the only reliable way to check permissions for a systemd service.

// func isAgentAccessibleBySystemd(agentPath string, username string) bool {
// 	// systemd-run creates a temporary service unit to run our command.
// 	// This ensures the command runs in the exact same security context as the real service.
// 	// cmd := exec.Command("systemd-run",
// 	// 	"--user="+username,
// 	// 	"--wait",
// 	// 	"--quiet",
// 	// 	"test", "-r", agentPath)

// 	cmd := exec.Command("systemd-run",
// 		"--property=User="+username, // ✅ CORRECT
// 		"--wait",
// 		"--quiet",
// 		"--service-type=oneshot", // ✅ ADD THIS
// 		"test", "-r", agentPath)

// 	// If this command returns an error (non-zero exit code), the test failed.
// 	// This accurately predicts that the real service would also fail to access the file.
// 	err := cmd.Run()
// 	return err == nil
// }

// // EnsureAgentInstalled checks if the agent exists and is properly configured
// func EnsureAgentInstalled(sourcePath string) (string, error) {
// 	// If source path is already the default location, just validate permissions
// 	if sourcePath == DefaultAgentPath {
// 		return DefaultAgentPath, ValidateAgentPermissions(DefaultAgentPath)
// 	}

// 	// Check if default agent already exists
// 	if _, err := os.Stat(DefaultAgentPath); err == nil {
// 		fmt.Printf("Agent already exists at %s\n", DefaultAgentPath)
// 		return DefaultAgentPath, ValidateAgentPermissions(DefaultAgentPath)
// 	}

// 	// Create directory structure
// 	if err := os.MkdirAll(DefaultAgentDir, 0o755); err != nil {
// 		return "", fmt.Errorf("failed to create agent directory: %w", err)
// 	}

// 	// Copy agent to default location
// 	if err := copyAgent(sourcePath, DefaultAgentPath); err != nil {
// 		return "", fmt.Errorf("failed to copy agent: %w", err)
// 	}

// 	// Set proper permissions
// 	if err := os.Chmod(DefaultAgentPath, 0o644); err != nil {
// 		return "", fmt.Errorf("failed to set agent permissions: %w", err)
// 	}

// 	// Set ownership to root:root
// 	if err := os.Chown(DefaultAgentPath, 0, 0); err != nil {
// 		return "", fmt.Errorf("failed to set agent ownership: %w", err)
// 	}

// 	fmt.Printf("✅ Agent installed to %s with proper permissions\n", DefaultAgentPath)
// 	return DefaultAgentPath, nil
// }

// // ValidateAgentPermissions ensures the agent JAR has correct permissions
// func ValidateAgentPermissions(agentPath string) error {
// 	info, err := os.Stat(agentPath)
// 	if err != nil {
// 		return fmt.Errorf("agent not found: %w", err)
// 	}

// 	// Check if world-readable
// 	mode := info.Mode()
// 	if mode&0o004 == 0 {
// 		fmt.Printf("⚠️  Warning: Agent is not world-readable\n")
// 		fmt.Printf("   Current permissions: %s\n", mode)
// 		fmt.Printf("   Fixing permissions...\n")

// 		if err := os.Chmod(agentPath, 0o644); err != nil {
// 			return fmt.Errorf("failed to fix permissions: %w", err)
// 		}
// 		fmt.Printf("✅ Permissions fixed to 0644\n")
// 	}

// 	return nil
// }

// // copyAgent copies the agent JAR from source to destination
// func copyAgent(src, dst string) error {
// 	source, err := os.Open(src)
// 	if err != nil {
// 		return err
// 	}
// 	defer source.Close()

// 	destination, err := os.Create(dst)
// 	if err != nil {
// 		return err
// 	}
// 	defer destination.Close()

// 	_, err = io.Copy(destination, source)
// 	return err
// }

// // CheckAgentAccessible tests if a user can access the agent
// func CheckAgentAccessible(agentPath, username string) bool {
// 	// This is a simplified check - for production, you'd want to
// 	// actually test with the specific user's permissions
// 	info, err := os.Stat(agentPath)
// 	if err != nil {
// 		return false
// 	}

// 	// Check if world-readable (0004 bit)
// 	return info.Mode()&0o004 != 0
// }
