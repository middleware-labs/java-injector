package systemd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// readExistingCatalinaOpts reads CATALINA_OPTS from the main service file
// Moved from main.go: readExistingCatalinaOpts()
func readExistingCatalinaOpts(serviceName string) string {
	// Try multiple possible locations
	possiblePaths := []string{
		fmt.Sprintf("/etc/systemd/system/%s.service", serviceName),
		fmt.Sprintf("/lib/systemd/system/%s.service", serviceName),
		fmt.Sprintf("/usr/lib/systemd/system/%s.service", serviceName),
	}

	for _, path := range possiblePaths {
		if opts := extractCatalinaOpts(path); opts != "" {
			return opts
		}
	}

	// Default if not found
	return "-Xms512M -Xmx1024M -server -XX:+UseParallelGC"
}

// extractCatalinaOpts extracts CATALINA_OPTS value from a service file
// Moved from main.go: extractCatalinaOpts()
func extractCatalinaOpts(serviceFile string) string {
	file, err := os.Open(serviceFile)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for: Environment="CATALINA_OPTS=..."
		if strings.HasPrefix(line, "Environment=\"CATALINA_OPTS=") {
			// Extract the value between quotes
			start := strings.Index(line, "CATALINA_OPTS=") + len("CATALINA_OPTS=")
			end := strings.LastIndex(line, "\"")
			if start > 0 && end > start {
				return line[start:end]
			}
		}
	}

	return ""
}

// GetTomcatServiceName returns the systemd service name for Tomcat
func GetTomcatServiceName() string {
	return "tomcat.service"
}

// IsTomcatService checks if a service name is a Tomcat service
func IsTomcatService(serviceName string) bool {
	return strings.Contains(strings.ToLower(serviceName), "tomcat")
}
