package systemd

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateDropIn creates a systemd drop-in file
// Moved from main.go: createSystemdDropIn()
func CreateDropIn(config *DropInConfig) error {
	// Read the config file to get actual values
	configVars, err := ReadConfigFile(config.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	var dropInContent string

	if config.IsTomcat {
		// Read existing CATALINA_OPTS from main service file
		existingOpts := readExistingCatalinaOpts(config.ServiceName)

		// Build full CATALINA_OPTS with agent appended
		fullOpts := fmt.Sprintf("%s -javaagent:%s", existingOpts, configVars["MW_JAVA_AGENT_PATH"])

		serviceNameWithHost := fmt.Sprintf("%s@%s", configVars["MW_SERVICE_NAME_PATTERN"], hostname)

		dropInContent = fmt.Sprintf(`[Service]
# Grant read access to Middleware agent
ReadOnlyPaths=%s

# Tomcat options with Middleware agent (hardcoded - systemd doesn't support variable expansion)
Environment="CATALINA_OPTS=%s"

# OpenTelemetry configuration
Environment="OTEL_SERVICE_NAME=%s"
Environment="OTEL_EXPORTER_OTLP_ENDPOINT=%s"
Environment="OTEL_EXPORTER_OTLP_HEADERS=authorization=%s"
Environment="OTEL_TRACES_EXPORTER=otlp"
Environment="OTEL_METRICS_EXPORTER=otlp"
Environment="OTEL_LOGS_EXPORTER=otlp"
`,
			configVars["MW_JAVA_AGENT_PATH"],
			fullOpts,
			serviceNameWithHost,
			configVars["MW_TARGET"],
			configVars["MW_API_KEY"])
	} else {
		dropInContent = fmt.Sprintf(`[Service]
Environment="JAVA_TOOL_OPTIONS=-javaagent:%s"
Environment="OTEL_SERVICE_NAME=%s"
Environment="OTEL_EXPORTER_OTLP_ENDPOINT=%s"
Environment="OTEL_EXPORTER_OTLP_HEADERS=authorization=%s"
Environment="OTEL_TRACES_EXPORTER=otlp"
Environment="OTEL_METRICS_EXPORTER=otlp"
Environment="OTEL_LOGS_EXPORTER=otlp"
`, configVars["MW_JAVA_AGENT_PATH"], configVars["MW_SERVICE_NAME"],
			configVars["MW_TARGET"], configVars["MW_API_KEY"])
	}

	// Create drop-in directory
	dropInDir := fmt.Sprintf("/etc/systemd/system/%s.d", config.ServiceName)
	if err := os.MkdirAll(dropInDir, 0o755); err != nil {
		return fmt.Errorf("failed to create drop-in directory: %v", err)
	}

	// Write drop-in file
	dropInPath := filepath.Join(dropInDir, "middleware-instrumentation.conf")
	if err := os.WriteFile(dropInPath, []byte(dropInContent), 0o644); err != nil {
		return fmt.Errorf("failed to write drop-in file: %v", err)
	}

	fmt.Printf("   Created drop-in: %s\n", dropInPath)
	return nil
}

// RemoveDropIn removes a systemd drop-in file
func RemoveDropIn(serviceName string) error {
	dropInDir := fmt.Sprintf("/etc/systemd/system/%s.d", serviceName)
	dropInPath := filepath.Join(dropInDir, "middleware-instrumentation.conf")

	if fileExists(dropInPath) {
		if err := os.Remove(dropInPath); err != nil {
			return fmt.Errorf("failed to remove drop-in file: %v", err)
		}
		fmt.Printf("   Removed drop-in: %s\n", dropInPath)

		// Remove directory if empty
		files, _ := os.ReadDir(dropInDir)
		if len(files) == 0 {
			if err := os.Remove(dropInDir); err == nil {
				fmt.Printf("   Removed empty directory: %s\n", dropInDir)
			}
		}
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
