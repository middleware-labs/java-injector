package naming

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/middleware-labs/java-injector/pkg/discovery"
)

// GenerateServiceName generates a service name for a Java process
// Moved from main.go: generateServiceName()
func GenerateServiceName(proc *discovery.JavaProcess) string {
	// For Tomcat services, use tomcat-{INSTANCE-NAME} pattern
	if proc.IsTomcat() {
		return GenerateForTomcat(proc)
	}

	return GenerateForStandard(proc)
}

// GenerateForTomcat generates service names for Tomcat processes
func GenerateForTomcat(proc *discovery.JavaProcess) string {
	tomcatInfo := proc.ExtractTomcatInfo()

	// Get instance name from CATALINA_BASE
	instanceName := filepath.Base(filepath.Dir(tomcatInfo.CatalinaBase))

	// Fallback: try CATALINA_BASE itself
	if instanceName == "." || instanceName == "/" || instanceName == "opt" {
		instanceName = filepath.Base(tomcatInfo.CatalinaBase)
	}

	// Clean the instance name (removes version numbers, apache- prefix, etc.)
	instanceName = CleanTomcatInstance(instanceName)

	// Handle edge cases
	if instanceName == "" || instanceName == "tomcat" {
		instanceName = "default"
	}

	// TODO: Once MW agent supports MW_SERVICE_NAME_PATTERN with {context} expansion,
	// we can return a pattern here instead of just the instance name.
	// For now, just return: tomcat-{instance}
	// Future: return pattern that agent will expand per-webapp
	return fmt.Sprintf("tomcat-%s", instanceName)
}

// GenerateForStandard generates service names for standard Java processes
func GenerateForStandard(proc *discovery.JavaProcess) string {
	// For non-Tomcat services, use JAR name as default
	if proc.JarFile != "" {
		cleaned := CleanJarName(proc.JarFile)
		if cleaned != "" {
			return cleaned
		}
	}

	// Last resort fallback
	if proc.ServiceName != "" && proc.ServiceName != "java-service" {
		cleaned := CleanServiceName(proc.ServiceName)
		if cleaned != "" {
			return cleaned
		}
	}

	return fmt.Sprintf("java-app-%d", proc.ProcessPID)
}

// GenerateWithOptions generates a service name with custom options
func GenerateWithOptions(proc *discovery.JavaProcess, opts ServiceNameOptions) string {
	// If a preferred name is provided, try to use it
	if opts.PreferredName != "" {
		cleaned := CleanServiceName(opts.PreferredName)
		if cleaned != "" && (opts.AllowGeneric || !IsGenericName(cleaned)) {
			return cleaned
		}
	}

	// Fall back to standard generation
	switch opts.ServiceType {
	case ServiceTypeTomcat:
		return GenerateForTomcat(proc)
	case ServiceTypeStandard:
		return GenerateForStandard(proc)
	default:
		return GenerateServiceName(proc)
	}
}

// ValidateServiceName checks if a service name meets naming requirements
func ValidateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Check for invalid characters
	if strings.ContainsAny(name, " _./\\:*?\"<>|") {
		return fmt.Errorf("service name contains invalid characters")
	}

	// Check if it's too generic
	if IsGenericName(name) {
		return fmt.Errorf("service name '%s' is too generic", name)
	}

	// Check length constraints
	if len(name) > 200 {
		return fmt.Errorf("service name is too long (max 200 characters)")
	}

	if len(name) < 1 {
		return fmt.Errorf("service name is too short")
	}

	return nil
}

// SuggestAlternativeName suggests an alternative name if the current one is invalid
func SuggestAlternativeName(proc *discovery.JavaProcess, invalidName string) string {
	// Try different generation strategies
	alternatives := []string{}

	// Try JAR-based name
	if proc.JarFile != "" {
		if alt := CleanJarName(proc.JarFile); alt != "" && alt != invalidName {
			alternatives = append(alternatives, alt)
		}
	}

	// Try service name
	if proc.ServiceName != "" && proc.ServiceName != "java-service" {
		if alt := CleanServiceName(proc.ServiceName); alt != "" && alt != invalidName {
			alternatives = append(alternatives, alt)
		}
	}

	// Return first valid alternative
	for _, alt := range alternatives {
		if ValidateServiceName(alt) == nil {
			return alt
		}
	}

	// Final fallback
	return fmt.Sprintf("java-service-%d", proc.ProcessPID)
}
