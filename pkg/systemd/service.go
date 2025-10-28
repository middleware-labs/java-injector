package systemd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/middleware-labs/java-injector/pkg/discovery"
	"github.com/middleware-labs/java-injector/pkg/naming"
)

// GetServiceName tries to find the actual systemd service name for a Java process
// Moved from main.go: getSystemdServiceName()
func GetServiceName(proc *discovery.JavaProcess) string {
	// Try to find the actual systemd service by PID
	cmd := exec.Command("systemctl", "status", fmt.Sprintf("%d", proc.ProcessPID))
	output, err := cmd.CombinedOutput()

	if err == nil {
		// Parse output to find service name
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			firstLine := lines[0]
			if strings.HasPrefix(firstLine, "●") || strings.HasPrefix(firstLine, "○") {
				parts := strings.Fields(firstLine)
				if len(parts) >= 2 {
					return parts[1]
				}
			}
		}
	}

	// Fallback: try common service name patterns
	serviceName := naming.GenerateServiceName(proc)
	possibleNames := []string{
		"spring-boot.service",
		serviceName + ".service",
		proc.ServiceName + ".service",
	}

	for _, name := range possibleNames {
		cmd := exec.Command("systemctl", "status", name)
		if err := cmd.Run(); err == nil {
			return name
		}
	}

	return serviceName + ".service"
}

// GetServiceStatus returns the status of a systemd service
func GetServiceStatus(serviceName string) (string, error) {
	cmd := exec.Command("systemctl", "is-active", serviceName)
	output, err := cmd.Output()
	if err != nil {
		return "unknown", err
	}

	return strings.TrimSpace(string(output)), nil
}

// RestartService restarts a systemd service
func RestartService(serviceName string) error {
	cmd := exec.Command("systemctl", "restart", serviceName)
	return cmd.Run()
}

// ReloadSystemd reloads the systemd daemon
func ReloadSystemd() error {
	cmd := exec.Command("systemctl", "daemon-reload")
	return cmd.Run()
}

// ServiceExists checks if a systemd service exists
func ServiceExists(serviceName string) bool {
	cmd := exec.Command("systemctl", "status", serviceName)
	err := cmd.Run()
	return err == nil
}
