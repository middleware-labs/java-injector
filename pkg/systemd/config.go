package systemd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CreateTomcatConfig creates configuration for Tomcat services
// Moved from main.go: createTomcatConfig()
func CreateTomcatConfig(configPath string, config *TomcatConfig) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	content := fmt.Sprintf(`# Middleware.io Configuration for Tomcat
# Instance: %s
# Generated: %s

# Dynamic service naming for webapps
MW_SERVICE_NAME_PATTERN=%s
MW_TOMCAT_INSTANCE=%s

# Middleware.io settings
MW_API_KEY=%s
MW_TARGET=%s
MW_LOG_LEVEL=INFO

# Java agent
MW_JAVA_AGENT_PATH=%s

# Telemetry collection
MW_APM_COLLECT_TRACES=true
MW_APM_COLLECT_METRICS=true
MW_APM_COLLECT_LOGS=true
`, config.InstanceName, getCurrentTime(), config.Pattern, config.InstanceName,
		config.APIKey, config.Target, config.AgentPath)

	return os.WriteFile(configPath, []byte(content), 0o644)
}

// CreateStandardConfig creates configuration for standard Java services
// Moved from main.go: createStandardConfig()
func CreateStandardConfig(configPath string, config *StandardConfig) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	content := fmt.Sprintf(`# Middleware.io Configuration
# Service: %s
# Generated: %s

# Service identification
MW_SERVICE_NAME=%s

# Middleware.io settings
MW_API_KEY=%s
MW_TARGET=%s
MW_LOG_LEVEL=INFO

# Java agent
MW_JAVA_AGENT_PATH=%s

# Telemetry collection
MW_APM_COLLECT_TRACES=true
MW_APM_COLLECT_METRICS=true
MW_APM_COLLECT_LOGS=true
`, config.ServiceName, getCurrentTime(), config.ServiceName,
		config.APIKey, config.Target, config.AgentPath)

	return os.WriteFile(configPath, []byte(content), 0o644)
}

// ReadConfigFile reads and parses a configuration file
// Moved from main.go: readConfigFile()
func ReadConfigFile(path string) (ConfigVars, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	vars := make(ConfigVars)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			vars[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return vars, nil
}

// getCurrentTime returns current time for config generation
func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
