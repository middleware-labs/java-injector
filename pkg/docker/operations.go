package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/middleware-labs/java-injector/pkg/config"
	"github.com/middleware-labs/java-injector/pkg/discovery"
)

const (
	// DefaultAgentPath is the default path to mount the agent in containers
	DefaultContainerAgentPath = "/opt/middleware/agents/middleware-javaagent.jar"

	// StateFile stores instrumented container information
	StateFile = "/etc/middleware/docker/instrumented.json"
)

// DockerOperations handles Docker container instrumentation operations
type DockerOperations struct {
	ctx           context.Context
	discoverer    *discovery.DockerDiscoverer
	hostAgentPath string
}

// NewDockerOperations creates a new Docker operations handler
func NewDockerOperations(ctx context.Context, hostAgentPath string) *DockerOperations {
	return &DockerOperations{
		ctx:           ctx,
		discoverer:    discovery.NewDockerDiscoverer(ctx),
		hostAgentPath: hostAgentPath,
	}
}

// InstrumentedState represents the state of instrumented containers
type InstrumentedState struct {
	Containers map[string]ContainerState `json:"containers"`
	UpdatedAt  time.Time                 `json:"updated_at"`
}

// ContainerState stores information about an instrumented container
type ContainerState struct {
	ContainerID    string            `json:"container_id"`
	ContainerName  string            `json:"container_name"`
	ImageName      string            `json:"image_name"`
	InstrumentedAt time.Time         `json:"instrumented_at"`
	AgentPath      string            `json:"agent_path"`
	OriginalEnv    map[string]string `json:"original_env"`
	ComposeFile    string            `json:"compose_file,omitempty"`
	ComposeService string            `json:"compose_service,omitempty"`

	RecreationCommand string `json:"recreation_command,omitempty"`
	OriginalConfig    string `json:"original_config,omitempty"`
}

// InstrumentContainer instruments a specific Docker container
func (do *DockerOperations) InstrumentContainer(containerName string, cfg *config.ProcessConfiguration) error {
	// Discover the container
	container, err := do.discoverer.GetContainerByName(containerName)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	// Check if already instrumented
	if container.Instrumented {
		return fmt.Errorf("container %s is already instrumented", containerName)
	}

	// Determine instrumentation strategy
	if container.IsCompose {
		return do.instrumentComposeContainer(container, cfg)
	}

	return do.instrumentStandaloneContainer(container, cfg)
}

// instrumentStandaloneContainer instruments a standalone Docker container
func (do *DockerOperations) instrumentStandaloneContainer(container *discovery.DockerContainer, cfg *config.ProcessConfiguration) error {
	fmt.Printf("🔧 Instrumenting standalone container: %s\n", container.ContainerName)

	// Step 1: Get and save original container configuration BEFORE making any changes
	containerConfig, err := do.getContainerConfig(container.ContainerID)
	if err != nil {
		return fmt.Errorf("failed to get container config: %w", err)
	}

	// Save original configuration as JSON string for restoration
	originalConfigBytes, err := json.Marshal(containerConfig)
	if err != nil {
		return fmt.Errorf("failed to serialize original config: %w", err)
	}

	// Build original recreation command from current state (before instrumentation)
	originalRecreationCommand := do.buildOriginalDockerRunCommand(containerConfig, container.ContainerName)

	// Step 2: Copy agent to container
	if err := do.copyAgentToContainer(container.ContainerID); err != nil {
		return fmt.Errorf("failed to copy agent: %w", err)
	}
	fmt.Println("   ✅ Agent copied to container")

	// Step 3: Build new environment variables with instrumentation
	newEnv := do.buildInstrumentationEnv(container, cfg)

	// Step 4: Stop the container
	fmt.Println("   🛑 Stopping container...")
	if err := do.stopContainer(container.ContainerID); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Step 5: Commit container to preserve any changes
	newImageName := fmt.Sprintf("%s-mw-instrumented:latest", container.ContainerName)
	if err := do.commitContainer(container.ContainerID, newImageName); err != nil {
		fmt.Printf("   ⚠️  Warning: Could not commit container: %v\n", err)
		// Use original image name if commit fails
		newImageName = container.ImageName + ":" + container.ImageTag
	}

	// Step 6: Remove old container
	if err := do.removeContainer(container.ContainerID); err != nil {
		return fmt.Errorf("failed to remove old container: %w", err)
	}

	// Step 7: Recreate container with instrumentation using the committed image
	instrumentedRunCommand := do.buildInstrumentedDockerRunCommand(containerConfig, newEnv, container.ContainerName, newImageName)
	if err := do.runContainer(instrumentedRunCommand); err != nil {
		return fmt.Errorf("failed to recreate container: %w", err)
	}

	// Step 8: Save state with ORIGINAL recreation command for proper restoration
	if err := do.saveContainerStateWithCommand(container, cfg, originalRecreationCommand, string(originalConfigBytes)); err != nil {
		fmt.Printf("   ⚠️  Warning: Could not save state: %v\n", err)
	}

	fmt.Printf("   ✅ Container %s instrumented successfully\n", container.ContainerName)
	return nil
}

// buildInstrumentedDockerRunCommand creates docker run command with instrumentation
func (do *DockerOperations) buildInstrumentedDockerRunCommand(config map[string]interface{}, env map[string]string, containerName, imageName string) string {
	var cmdParts []string
	cmdParts = append(cmdParts, "docker", "run", "-d")
	cmdParts = append(cmdParts, "--name", containerName)

	// Add environment variables (with instrumentation)
	for k, v := range env {
		cmdParts = append(cmdParts, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add original volume mounts
	if mounts, ok := config["Mounts"].([]interface{}); ok {
		for _, m := range mounts {
			if mount, ok := m.(map[string]interface{}); ok {
				src, srcOk := mount["Source"].(string)
				dst, dstOk := mount["Destination"].(string)

				if srcOk && dstOk {
					mode := "rw"
					if rw, ok := mount["RW"].(bool); ok && !rw {
						mode = "ro"
					}
					cmdParts = append(cmdParts, "-v", fmt.Sprintf("%s:%s:%s", src, dst, mode))
				}
			}
		}
	}

	// Add agent volume mount
	cmdParts = append(cmdParts, "-v", fmt.Sprintf("%s:%s:ro", do.hostAgentPath, DefaultContainerAgentPath))

	// Add port mappings
	if networkSettings, ok := config["NetworkSettings"].(map[string]interface{}); ok {
		if ports, ok := networkSettings["Ports"].(map[string]interface{}); ok {
			for containerPort, bindings := range ports {
				if bindingList, ok := bindings.([]interface{}); ok && len(bindingList) > 0 {
					if binding, ok := bindingList[0].(map[string]interface{}); ok {
						if hostPort, ok := binding["HostPort"].(string); ok && hostPort != "" {
							hostIP := "0.0.0.0"
							if hip, ok := binding["HostIp"].(string); ok && hip != "" {
								hostIP = hip
							}
							cmdParts = append(cmdParts, "-p", fmt.Sprintf("%s:%s:%s", hostIP, hostPort, containerPort))
						}
					}
				}
			}
		}
	}

	// Add networks
	if networkSettings, ok := config["NetworkSettings"].(map[string]interface{}); ok {
		if networks, ok := networkSettings["Networks"].(map[string]interface{}); ok {
			for networkName := range networks {
				if networkName != "bridge" {
					cmdParts = append(cmdParts, "--network", networkName)
				}
			}
		}
	}

	// Add restart policy
	if hostConfig, ok := config["HostConfig"].(map[string]interface{}); ok {
		if restartPolicy, ok := hostConfig["RestartPolicy"].(map[string]interface{}); ok {
			if name, ok := restartPolicy["Name"].(string); ok && name != "" && name != "no" {
				if maxRetries, ok := restartPolicy["MaximumRetryCount"].(float64); ok && maxRetries > 0 {
					cmdParts = append(cmdParts, "--restart", fmt.Sprintf("%s:%d", name, int(maxRetries)))
				} else {
					cmdParts = append(cmdParts, "--restart", name)
				}
			}
		}

		// Add working directory
		if configSection, ok := config["Config"].(map[string]interface{}); ok {
			if workingDir, ok := configSection["WorkingDir"].(string); ok && workingDir != "" {
				cmdParts = append(cmdParts, "--workdir", workingDir)
			}

			// Add user
			if user, ok := configSection["User"].(string); ok && user != "" {
				cmdParts = append(cmdParts, "--user", user)
			}

			// Add original command
			if cmd, ok := configSection["Cmd"].([]interface{}); ok && len(cmd) > 0 {
				cmdParts = append(cmdParts, imageName)
				for _, c := range cmd {
					if cStr, ok := c.(string); ok {
						cmdParts = append(cmdParts, cStr)
					}
				}
				return strings.Join(cmdParts, " ")
			}
		}
	}

	// Add image
	cmdParts = append(cmdParts, imageName)

	return strings.Join(cmdParts, " ")
}

// instrumentComposeContainer instruments a Docker Compose-managed container
func (do *DockerOperations) instrumentComposeContainer(container *discovery.DockerContainer, cfg *config.ProcessConfiguration) error {
	fmt.Printf("🔧 Instrumenting Docker Compose container: %s\n", container.ContainerName)

	if container.ComposeFile == "" {
		return fmt.Errorf("compose file not found for container %s", container.ContainerName)
	}

	// Step 1: Backup original compose file
	backupFile := container.ComposeFile + ".backup"
	if err := do.copyFile(container.ComposeFile, backupFile); err != nil {
		fmt.Printf("   ⚠️  Warning: Could not backup compose file: %v\n", err)
	}

	// Step 2: Modify compose file
	if err := do.modifyComposeFile(container, cfg); err != nil {
		return fmt.Errorf("failed to modify compose file: %w", err)
	}
	fmt.Println("   ✅ Compose file updated")

	// Step 3: Recreate service using docker-compose
	fmt.Println("   🔄 Recreating service...")
	if err := do.recreateComposeService(container); err != nil {
		// Restore backup on failure
		do.copyFile(backupFile, container.ComposeFile)
		return fmt.Errorf("failed to recreate service: %w", err)
	}

	// Step 4: Save state
	if err := do.saveContainerState(container, cfg); err != nil {
		fmt.Printf("   ⚠️  Warning: Could not save state: %v\n", err)
	}

	fmt.Printf("   ✅ Container %s instrumented successfully\n", container.ContainerName)
	return nil
}

// buildOriginalDockerRunCommand creates the original docker run command before instrumentation
func (do *DockerOperations) buildOriginalDockerRunCommand(config map[string]interface{}, containerName string) string {
	var cmdParts []string
	cmdParts = append(cmdParts, "docker", "run", "-d")
	cmdParts = append(cmdParts, "--name", containerName)

	configSection, ok := config["Config"].(map[string]interface{})
	if !ok {
		return ""
	}

	// Add original environment variables (without instrumentation)
	if env, ok := configSection["Env"].([]interface{}); ok {
		for _, e := range env {
			if envStr, ok := e.(string); ok {
				// Skip any existing MW_ or OTEL_ variables and JAVA_TOOL_OPTIONS with javaagent
				if !strings.HasPrefix(envStr, "MW_") &&
					!strings.HasPrefix(envStr, "OTEL_") &&
					!(strings.HasPrefix(envStr, "JAVA_TOOL_OPTIONS=") && strings.Contains(envStr, "javaagent")) {
					cmdParts = append(cmdParts, "-e", envStr)
				}
			}
		}
	}

	// Add original volume mounts (excluding our agent mount)
	if mounts, ok := config["Mounts"].([]interface{}); ok {
		for _, m := range mounts {
			if mount, ok := m.(map[string]interface{}); ok {
				src, srcOk := mount["Source"].(string)
				dst, dstOk := mount["Destination"].(string)

				if srcOk && dstOk {
					// Skip our agent mount
					if dst == DefaultContainerAgentPath {
						continue
					}

					mode := "rw"
					if rw, ok := mount["RW"].(bool); ok && !rw {
						mode = "ro"
					}
					cmdParts = append(cmdParts, "-v", fmt.Sprintf("%s:%s:%s", src, dst, mode))
				}
			}
		}
	}

	// Add port mappings
	if networkSettings, ok := config["NetworkSettings"].(map[string]interface{}); ok {
		if ports, ok := networkSettings["Ports"].(map[string]interface{}); ok {
			for containerPort, bindings := range ports {
				if bindingList, ok := bindings.([]interface{}); ok && len(bindingList) > 0 {
					if binding, ok := bindingList[0].(map[string]interface{}); ok {
						if hostPort, ok := binding["HostPort"].(string); ok && hostPort != "" {
							hostIP := "0.0.0.0"
							if hip, ok := binding["HostIp"].(string); ok && hip != "" {
								hostIP = hip
							}
							cmdParts = append(cmdParts, "-p", fmt.Sprintf("%s:%s:%s", hostIP, hostPort, containerPort))
						}
					}
				}
			}
		}
	}

	// Add networks
	if networkSettings, ok := config["NetworkSettings"].(map[string]interface{}); ok {
		if networks, ok := networkSettings["Networks"].(map[string]interface{}); ok {
			for networkName := range networks {
				if networkName != "bridge" { // Skip default bridge network
					cmdParts = append(cmdParts, "--network", networkName)
				}
			}
		}
	}

	// Add restart policy
	if hostConfig, ok := config["HostConfig"].(map[string]interface{}); ok {
		if restartPolicy, ok := hostConfig["RestartPolicy"].(map[string]interface{}); ok {
			if name, ok := restartPolicy["Name"].(string); ok && name != "" && name != "no" {
				if maxRetries, ok := restartPolicy["MaximumRetryCount"].(float64); ok && maxRetries > 0 {
					cmdParts = append(cmdParts, "--restart", fmt.Sprintf("%s:%d", name, int(maxRetries)))
				} else {
					cmdParts = append(cmdParts, "--restart", name)
				}
			}
		}

		// Add working directory
		if workingDir, ok := configSection["WorkingDir"].(string); ok && workingDir != "" {
			cmdParts = append(cmdParts, "--workdir", workingDir)
		}

		// Add user
		if user, ok := configSection["User"].(string); ok && user != "" {
			cmdParts = append(cmdParts, "--user", user)
		}
	}

	// Add original image
	if image, ok := configSection["Image"].(string); ok {
		cmdParts = append(cmdParts, image)
	}

	// Add original command
	if cmd, ok := configSection["Cmd"].([]interface{}); ok && len(cmd) > 0 {
		for _, c := range cmd {
			if cStr, ok := c.(string); ok {
				cmdParts = append(cmdParts, cStr)
			}
		}
	}

	return strings.Join(cmdParts, " ")
}

// saveContainerStateWithCommand saves container state with recreation command
func (do *DockerOperations) saveContainerStateWithCommand(container *discovery.DockerContainer, cfg *config.ProcessConfiguration, recreationCommand, originalConfig string) error {
	state, _ := do.loadState()
	if state.Containers == nil {
		state.Containers = make(map[string]ContainerState)
	}

	state.Containers[container.ContainerName] = ContainerState{
		ContainerID:       container.ContainerID,
		ContainerName:     container.ContainerName,
		ImageName:         container.ImageName,
		InstrumentedAt:    time.Now(),
		AgentPath:         do.hostAgentPath,
		OriginalEnv:       container.Environment,
		ComposeFile:       container.ComposeFile,
		ComposeService:    container.ComposeService,
		RecreationCommand: recreationCommand, // Now properly set!
		OriginalConfig:    originalConfig,    // Full original config for debugging
	}
	state.UpdatedAt = time.Now()

	return do.saveState(state)
}

// UninstrumentContainer removes instrumentation from a container
func (do *DockerOperations) UninstrumentContainer(containerName string) error {
	// Load state to check if container was instrumented by us
	state, err := do.loadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	containerState, exists := state.Containers[containerName]
	if !exists {
		return fmt.Errorf("container %s was not instrumented by this tool", containerName)
	}

	fmt.Printf("🔧 Uninstrumenting container: %s\n", containerName)

	// Check if it's a compose container
	if containerState.ComposeFile != "" {
		return do.uninstrumentComposeContainer(&containerState)
	}

	return do.uninstrumentStandaloneContainer(&containerState)
}

// uninstrumentStandaloneContainer removes instrumentation from standalone container
func (do *DockerOperations) uninstrumentStandaloneContainer(state *ContainerState) error {
	// Check if we have the original recreation command
	if state.RecreationCommand == "" {
		fmt.Println("   ⚠️  Cannot fully restore container without original configuration")
		fmt.Println("   💡 Suggestion: Remove JAVA_TOOL_OPTIONS and MW_* env vars manually and restart")
		return do.removeContainerState(state.ContainerName)
	}

	fmt.Println("   🔄 Restoring original container configuration...")

	// Stop current container
	if err := do.stopContainerByName(state.ContainerName); err != nil {
		fmt.Printf("   ⚠️  Warning: Could not stop container: %v\n", err)
	}

	// Remove current container
	if err := do.removeContainerByName(state.ContainerName); err != nil {
		fmt.Printf("   ⚠️  Warning: Could not remove container: %v\n", err)
	}

	// Recreate with original command
	fmt.Printf("   Executing: %s\n", state.RecreationCommand)
	if err := do.runContainer(state.RecreationCommand); err != nil {
		return fmt.Errorf("failed to recreate container with original config: %w", err)
	}

	fmt.Printf("   ✅ Container %s restored to original configuration\n", state.ContainerName)

	// Remove from state
	return do.removeContainerState(state.ContainerName)
}

// uninstrumentComposeContainer removes instrumentation from compose container
func (do *DockerOperations) uninstrumentComposeContainer(state *ContainerState) error {
	// Restore backup compose file
	backupFile := state.ComposeFile + ".backup"
	if _, err := os.Stat(backupFile); err == nil {
		if err := do.copyFile(backupFile, state.ComposeFile); err != nil {
			return fmt.Errorf("failed to restore compose file: %w", err)
		}
		fmt.Println("   ✅ Compose file restored")

		// Recreate service
		fmt.Println("   🔄 Recreating service...")

		// Get container to recreate
		container, err := do.discoverer.GetContainerByName(state.ContainerName)
		if err == nil {
			do.recreateComposeService(container)
		}
	} else {
		fmt.Println("   ⚠️  Backup compose file not found")
		fmt.Println("   💡 Suggestion: Manually remove MW instrumentation from compose file and run 'docker-compose up -d'")
	}

	// Remove from state
	return do.removeContainerState(state.ContainerName)
}

// copyAgentToContainer copies the agent JAR to a running container
func (do *DockerOperations) copyAgentToContainer(containerID string) error {
	// Create directory in container
	mkdirCmd := exec.CommandContext(do.ctx, "docker", "exec", containerID, "mkdir", "-p", "/opt/middleware/agents")
	if err := mkdirCmd.Run(); err != nil {
		// Try without mkdir if it fails (some distroless images don't have mkdir)
		fmt.Println("   ⚠️  Could not create directory, trying direct copy...")
	}

	// Copy agent file
	containerPath := containerID + ":" + DefaultContainerAgentPath
	cmd := exec.CommandContext(do.ctx, "docker", "cp", do.hostAgentPath, containerPath)
	return cmd.Run()
}

// buildInstrumentationEnv builds environment variables for instrumentation
func (do *DockerOperations) buildInstrumentationEnv(container *discovery.DockerContainer, cfg *config.ProcessConfiguration) map[string]string {
	env := make(map[string]string)

	// Copy existing environment
	for k, v := range container.Environment {
		env[k] = v
	}

	// Add JAVA_TOOL_OPTIONS
	javaToolOptions := fmt.Sprintf("-javaagent:%s", DefaultContainerAgentPath)
	if existing, ok := env["JAVA_TOOL_OPTIONS"]; ok {
		// Append to existing
		env["JAVA_TOOL_OPTIONS"] = existing + " " + javaToolOptions
	} else {
		env["JAVA_TOOL_OPTIONS"] = javaToolOptions
	}

	// Add MW configuration
	mwEnv := cfg.ToEnvironmentVariables()
	for k, v := range mwEnv {
		env[k] = v
	}

	// Set service name if not already set
	if env["MW_SERVICE_NAME"] == "" {
		env["MW_SERVICE_NAME"] = container.GetServiceName()
	}

	return env
}

// getContainerConfig gets full container configuration
func (do *DockerOperations) getContainerConfig(containerID string) (map[string]interface{}, error) {
	cmd := exec.CommandContext(do.ctx, "docker", "inspect", containerID)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var inspectData []map[string]interface{}
	if err := json.Unmarshal(output, &inspectData); err != nil {
		return nil, err
	}

	if len(inspectData) == 0 {
		return nil, fmt.Errorf("no data returned")
	}

	return inspectData[0], nil
}

// stopContainer stops a running container
func (do *DockerOperations) stopContainer(containerID string) error {
	cmd := exec.CommandContext(do.ctx, "docker", "stop", containerID)
	return cmd.Run()
}

// stopContainerByName stops a container by name
func (do *DockerOperations) stopContainerByName(name string) error {
	cmd := exec.CommandContext(do.ctx, "docker", "stop", name)
	return cmd.Run()
}

// removeContainer removes a container
func (do *DockerOperations) removeContainer(containerID string) error {
	cmd := exec.CommandContext(do.ctx, "docker", "rm", containerID)
	return cmd.Run()
}

// removeContainerByName removes a container by name
func (do *DockerOperations) removeContainerByName(name string) error {
	cmd := exec.CommandContext(do.ctx, "docker", "rm", name)
	return cmd.Run()
}

// commitContainer commits a container to a new image
func (do *DockerOperations) commitContainer(containerID, imageName string) error {
	cmd := exec.CommandContext(do.ctx, "docker", "commit", containerID, imageName)
	return cmd.Run()
}

// buildDockerRunCommand builds a docker run command from container config
func (do *DockerOperations) buildDockerRunCommand(config map[string]interface{}, env map[string]string, containerName string) string {
	var cmdParts []string
	cmdParts = append(cmdParts, "docker", "run", "-d")
	cmdParts = append(cmdParts, "--name", containerName)

	// Add environment variables
	for k, v := range env {
		cmdParts = append(cmdParts, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Add volume mounts
	if mounts, ok := config["Mounts"].([]interface{}); ok {
		for _, m := range mounts {
			if mount, ok := m.(map[string]interface{}); ok {
				src := mount["Source"].(string)
				dst := mount["Destination"].(string)
				mode := "rw"
				if rw, ok := mount["RW"].(bool); ok && !rw {
					mode = "ro"
				}
				cmdParts = append(cmdParts, "-v", fmt.Sprintf("%s:%s:%s", src, dst, mode))
			}
		}
	}

	// Add host agent path as volume
	cmdParts = append(cmdParts, "-v", fmt.Sprintf("%s:%s:ro", do.hostAgentPath, DefaultContainerAgentPath))

	// Add ports
	if networkSettings, ok := config["NetworkSettings"].(map[string]interface{}); ok {
		if ports, ok := networkSettings["Ports"].(map[string]interface{}); ok {
			for containerPort, bindings := range ports {
				if bindingList, ok := bindings.([]interface{}); ok && len(bindingList) > 0 {
					if binding, ok := bindingList[0].(map[string]interface{}); ok {
						hostPort := binding["HostPort"]
						cmdParts = append(cmdParts, "-p", fmt.Sprintf("%v:%v", hostPort, containerPort))
					}
				}
			}
		}
	}

	// Add image
	if configSection, ok := config["Config"].(map[string]interface{}); ok {
		if image, ok := configSection["Image"].(string); ok {
			cmdParts = append(cmdParts, image)
		}
	}

	return strings.Join(cmdParts, " ")
}

// runContainer runs a docker run command
func (do *DockerOperations) runContainer(command string) error {
	cmd := exec.CommandContext(do.ctx, "sh", "-c", command)
	return cmd.Run()
}

// modifyComposeFile modifies a docker-compose.yml file to add instrumentation
func (do *DockerOperations) modifyComposeFile(container *discovery.DockerContainer, cfg *config.ProcessConfiguration) error {
	// Read compose file
	content, err := os.ReadFile(container.ComposeFile)
	if err != nil {
		return err
	}

	// For now, we'll do simple string manipulation
	// In production, you'd want to use a YAML parser
	newContent := string(content)

	// Find the service section
	serviceName := container.ComposeService
	servicePattern := fmt.Sprintf("  %s:", serviceName)

	if !strings.Contains(newContent, servicePattern) {
		return fmt.Errorf("service %s not found in compose file", serviceName)
	}

	// Build environment additions
	envAdditions := ""
	mwEnv := cfg.ToEnvironmentVariables()

	envAdditions += "      - JAVA_TOOL_OPTIONS=-javaagent:" + DefaultContainerAgentPath + "\n"
	for k, v := range mwEnv {
		envAdditions += fmt.Sprintf("      - %s=%s\n", k, v)
	}

	// Add volume for agent
	volumeAddition := fmt.Sprintf("      - %s:%s:ro\n", do.hostAgentPath, DefaultContainerAgentPath)

	// Insert additions (this is simplified - production code should use YAML parser)
	fmt.Println("   ⚠️  Compose file modification requires manual intervention")
	fmt.Println("   💡 Add the following to your docker-compose.yml:")
	fmt.Println("\n   Environment variables:")
	fmt.Println(envAdditions)
	fmt.Println("   Volume mount:")
	fmt.Println(volumeAddition)

	return nil
}

// recreateComposeService recreates a Docker Compose service
func (do *DockerOperations) recreateComposeService(container *discovery.DockerContainer) error {
	if container.ComposeWorkDir == "" {
		return fmt.Errorf("compose working directory not found")
	}

	// Change to compose directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	if err := os.Chdir(container.ComposeWorkDir); err != nil {
		return err
	}

	// Run docker-compose up -d for specific service
	cmd := exec.CommandContext(do.ctx, "docker-compose", "up", "-d", container.ComposeService)
	return cmd.Run()
}

// saveContainerState saves instrumented container state
func (do *DockerOperations) saveContainerState(container *discovery.DockerContainer, cfg *config.ProcessConfiguration) error {
	state, _ := do.loadState()
	if state.Containers == nil {
		state.Containers = make(map[string]ContainerState)
	}

	state.Containers[container.ContainerName] = ContainerState{
		ContainerID:    container.ContainerID,
		ContainerName:  container.ContainerName,
		ImageName:      container.ImageName,
		InstrumentedAt: time.Now(),
		AgentPath:      do.hostAgentPath,
		OriginalEnv:    container.Environment,
		ComposeFile:    container.ComposeFile,
		ComposeService: container.ComposeService,
	}
	state.UpdatedAt = time.Now()

	return do.saveState(state)
}

// removeContainerState removes container from state
func (do *DockerOperations) removeContainerState(containerName string) error {
	state, _ := do.loadState()
	delete(state.Containers, containerName)
	state.UpdatedAt = time.Now()
	return do.saveState(state)
}

// loadState loads the instrumented containers state
func (do *DockerOperations) loadState() (*InstrumentedState, error) {
	if _, err := os.Stat(StateFile); os.IsNotExist(err) {
		return &InstrumentedState{
			Containers: make(map[string]ContainerState),
		}, nil
	}

	data, err := os.ReadFile(StateFile)
	if err != nil {
		return nil, err
	}

	var state InstrumentedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// saveState saves the instrumented containers state
func (do *DockerOperations) saveState(state *InstrumentedState) error {
	// Ensure directory exists
	dir := filepath.Dir(StateFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(StateFile, data, 0o644)
}

// copyFile copies a file from src to dst
func (do *DockerOperations) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// ListInstrumentedContainers lists all instrumented containers
func (do *DockerOperations) ListInstrumentedContainers() ([]ContainerState, error) {
	state, err := do.loadState()
	if err != nil {
		return nil, err
	}

	var containers []ContainerState
	for _, c := range state.Containers {
		containers = append(containers, c)
	}

	return containers, nil
}
