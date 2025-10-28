package naming

import (
	"path/filepath"
	"regexp"
	"strings"
)

// CleanServiceName applies final cleaning to service name
// Moved from main.go: cleanServiceName()
func CleanServiceName(name string) string {
	if name == "" {
		return ""
	}

	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace underscores with hyphens
	name = strings.ReplaceAll(name, "_", "-")

	// Remove invalid characters (keep only alphanumeric and hyphens)
	reg := regexp.MustCompile(`[^a-z0-9\-]+`)
	name = reg.ReplaceAllString(name, "")

	// Remove leading/trailing hyphens
	name = strings.Trim(name, "-")

	// Collapse multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	name = reg.ReplaceAllString(name, "-")

	// Final validation - must not be empty and not be generic
	if name == "" || IsGenericName(name) {
		return ""
	}

	return name
}

// CleanTomcatInstance cleans Tomcat instance names
// Moved from main.go: cleanTomcatInstance()
func CleanTomcatInstance(name string) string {
	name = filepath.Base(name)
	name = strings.TrimPrefix(name, "apache-")
	re := regexp.MustCompile(`-\d+\.\d+\.\d+.*$`)
	name = re.ReplaceAllString(name, "")

	return CleanServiceName(name)
}

// CleanJarName cleans JAR file names for use as service names
// Moved from main.go: cleanJarName()
func CleanJarName(jar string) string {
	name := strings.TrimSuffix(jar, ".jar")
	patterns := []string{
		`-\d+\.\d+\.\d+.*$`,
		`-SNAPSHOT$`,
		`_\d+\.\d+\.\d+.*$`,
		`-BUILD-\d+$`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		name = re.ReplaceAllString(name, "")
	}
	return CleanServiceName(name)
}

// CleanWebappName cleans webapp names
// Moved from main.go: cleanWebappName()
func CleanWebappName(name string) string {
	return CleanJarName(name)
}

// IsGenericName checks if a service name is too generic
// Moved from main.go: isGenericServiceName()
func IsGenericName(name string) bool {
	genericNames := []string{
		"java", "app", "application", "service", "server", "main",
		"demo", "test", "example", "sample", "hello", "world",
	}

	nameLower := strings.ToLower(name)
	for _, generic := range genericNames {
		if nameLower == generic {
			return true
		}
	}

	return false
}

// CamelToKebab converts CamelCase to kebab-case
// This is a utility that can be used by discovery package if needed
func CamelToKebab(s string) string {
	// Insert hyphens before uppercase letters (except the first character)
	reg := regexp.MustCompile(`([a-z])([A-Z])`)
	s = reg.ReplaceAllString(s, "${1}-${2}")

	return strings.ToLower(s)
}

// NormalizeServiceName normalizes a service name according to common conventions
// This provides a standard normalization that other packages can use
func NormalizeServiceName(name string) string {
	if name == "" {
		return ""
	}

	// Convert to lowercase
	name = strings.ToLower(name)

	// Replace underscores and spaces with hyphens
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")

	// Remove invalid characters (keep only alphanumeric and hyphens)
	reg := regexp.MustCompile(`[^a-z0-9\-]+`)
	name = reg.ReplaceAllString(name, "")

	// Remove leading/trailing hyphens
	name = strings.Trim(name, "-")

	// Collapse multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	name = reg.ReplaceAllString(name, "-")

	return name
}
