// Package javanaming provides Java-specific service name generation with
// sanitization rules. It derives a human-readable service name from a
// discovered Java process using JAR filename cleaning, Tomcat instance
// naming, and generic name filtering.
package javanaming

import (
	"fmt"
	"path/filepath"

	"github.com/middleware-labs/mw-injector/pkg/discovery"
)

// GenerateServiceName generates a service name for a Java process.
func GenerateServiceName(proc *discovery.Process) string {
	if proc.DetailBool(discovery.DetailIsTomcat) {
		return GenerateForTomcat(proc)
	}

	return GenerateForStandard(proc)
}

// GenerateForTomcat generates service names for Tomcat processes.
func GenerateForTomcat(proc *discovery.Process) string {
	tomcatInfo := proc.ExtractTomcatInfo()

	// Get instance name from CATALINA_BASE
	instanceName := filepath.Base(filepath.Dir(tomcatInfo.CatalinaBase))

	// Fallback: try CATALINA_BASE itself
	if instanceName == "." || instanceName == "/" || instanceName == "opt" {
		instanceName = filepath.Base(tomcatInfo.CatalinaBase)
	}

	instanceName = CleanTomcatInstance(instanceName)

	if instanceName == "" || instanceName == "tomcat" {
		instanceName = "default"
	}

	return fmt.Sprintf("tomcat-%s", instanceName)
}

// GenerateForStandard generates service names for standard Java processes.
func GenerateForStandard(proc *discovery.Process) string {
	jarFile := proc.DetailString(discovery.DetailJarFile)
	if jarFile != "" {
		cleaned := CleanJarName(jarFile)
		if cleaned != "" {
			return cleaned
		}
	}

	if proc.ServiceName != "" && proc.ServiceName != "java-service" {
		cleaned := CleanServiceName(proc.ServiceName)
		if cleaned != "" {
			return cleaned
		}
	}

	return fmt.Sprintf("java-app-%d", proc.PID)
}

