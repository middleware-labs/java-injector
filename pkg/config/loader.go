package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Middleware      MiddlewareConfig      `yaml:"middleware"`
	Agent           AgentConfig           `yaml:"agent"`
	Service         ServiceConfig         `yaml:"service"`
	Host            HostConfig            `yaml:"host"`
	Instrumentation InstrumentationConfig `yaml:"instrumentation"`
	Docker          DockerConfig          `yaml:"docker"`
	Logging         LoggingConfig         `yaml:"logging"`
	Advanced        AdvancedConfig        `yaml:"advanced"`
}

// MiddlewareConfig holds Middleware.io connection settings
type MiddlewareConfig struct {
	APIKey string `yaml:"api_key" validate:"required"`
	Target string `yaml:"target" validate:"required"`
}

// AgentConfig holds Java agent configuration
type AgentConfig struct {
	Path         string `yaml:"path"`
	Version      string `yaml:"version"`
	AutoDownload bool   `yaml:"auto_download"`
}

// ServiceConfig holds service naming and tagging configuration
type ServiceConfig struct {
	NamePrefix  string            `yaml:"name_prefix"`
	NameSuffix  string            `yaml:"name_suffix"`
	Environment string            `yaml:"environment"`
	Tags        map[string]string `yaml:"tags"`
}

// HostConfig holds host-specific configuration
type HostConfig struct {
	Hostname        string `yaml:"hostname"`
	IncludeHostname bool   `yaml:"include_hostname"`
}

// InstrumentationConfig holds instrumentation behavior settings
type InstrumentationConfig struct {
	AutoConfirm        bool                     `yaml:"auto_confirm"`
	SkipInstrumented   bool                     `yaml:"skip_instrumented"`
	RestartServices    bool                     `yaml:"restart_services"`
	PermissionHandling PermissionHandlingConfig `yaml:"permission_handling"`
}

// PermissionHandlingConfig holds permission handling settings
type PermissionHandlingConfig struct {
	AutoCopyAgent        bool `yaml:"auto_copy_agent"`
	SkipPermissionErrors bool `yaml:"skip_permission_errors"`
}

// DockerConfig holds Docker-specific configuration
type DockerConfig struct {
	Network    string           `yaml:"network"`
	AgentMount AgentMountConfig `yaml:"agent_mount"`
}

// AgentMountConfig holds Docker volume mount configuration
type AgentMountConfig struct {
	HostPath      string `yaml:"host_path"`
	ContainerPath string `yaml:"container_path"`
	Mode          string `yaml:"mode"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
	JSON  bool   `yaml:"json"`
}

// AdvancedConfig holds advanced configuration options
type AdvancedConfig struct {
	Timeout        int    `yaml:"timeout"`
	MaxConcurrency int    `yaml:"max_concurrency"`
	StateDirectory string `yaml:"state_directory"`
	BackupConfigs  bool   `yaml:"backup_configs"`
}

// DefaultConfig returns configuration with sensible defaults
func DefaultConfig() *Config {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	return &Config{
		Middleware: MiddlewareConfig{
			Target: "https://prod.middleware.io:443",
		},
		Agent: AgentConfig{
			Path:         "/opt/middleware/agents/middleware-javaagent-1.8.1.jar",
			AutoDownload: false,
		},
		Service: ServiceConfig{
			Environment: "production",
			Tags:        make(map[string]string),
		},
		Host: HostConfig{
			Hostname:        hostname,
			IncludeHostname: false,
		},
		Instrumentation: InstrumentationConfig{
			AutoConfirm:      false,
			SkipInstrumented: false,
			RestartServices:  true,
			PermissionHandling: PermissionHandlingConfig{
				AutoCopyAgent:        true,
				SkipPermissionErrors: false,
			},
		},
		Docker: DockerConfig{
			AgentMount: AgentMountConfig{
				HostPath:      "/opt/middleware/agents",
				ContainerPath: "/opt/middleware/agents",
				Mode:          "ro",
			},
		},
		Logging: LoggingConfig{
			Level: "INFO",
			JSON:  false,
		},
		Advanced: AdvancedConfig{
			Timeout:        300,
			MaxConcurrency: 10,
			StateDirectory: "/etc/middleware/state",
			BackupConfigs:  true,
		},
	}
}

// LoadConfig loads configuration from file with fallback locations
func LoadConfig() (*Config, error) {
	// Configuration file search paths (in order of preference)
	configPaths := []string{
		"./mw-injector.yaml",                    // Current directory
		"./mw-injector.yml",                     // Current directory (alternative extension)
		"/etc/middleware/mw-injector.yaml",      // System-wide configuration
		"/etc/middleware/mw-injector.yml",       // System-wide (alternative extension)
		os.ExpandEnv("$HOME/.mw-injector.yaml"), // User-specific configuration
		os.ExpandEnv("$HOME/.mw-injector.yml"),  // User-specific (alternative extension)
	}

	// Check for config file specified via environment variable
	if envConfigPath := os.Getenv("MW_INJECTOR_CONFIG"); envConfigPath != "" {
		// Prepend env config path to search paths
		configPaths = append([]string{envConfigPath}, configPaths...)
	}

	// Start with default configuration
	config := DefaultConfig()

	// Try to load from each path
	var configFileUsed string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			if err := loadConfigFromFile(path, config); err != nil {
				return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
			}
			configFileUsed = path
			break
		}
	}

	// Override with environment variables if present
	if err := loadConfigFromEnv(config); err != nil {
		return nil, fmt.Errorf("failed to load config from environment: %w", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Log which config file was used (if any)
	if configFileUsed != "" {
		fmt.Printf("📋 Loaded configuration from: %s\n", configFileUsed)
	} else {
		fmt.Println("📋 Using default configuration (no config file found)")
	}

	return config, nil
}

// loadConfigFromFile loads configuration from a YAML file
func loadConfigFromFile(path string, config *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	return nil
}

// loadConfigFromEnv loads configuration from environment variables
func loadConfigFromEnv(config *Config) error {
	// Middleware configuration
	if apiKey := os.Getenv("MW_API_KEY"); apiKey != "" {
		config.Middleware.APIKey = apiKey
	}
	if target := os.Getenv("MW_TARGET"); target != "" {
		config.Middleware.Target = target
	}

	// Agent configuration
	if agentPath := os.Getenv("MW_AGENT_PATH"); agentPath != "" {
		config.Agent.Path = agentPath
	}

	// Service configuration
	if env := os.Getenv("MW_ENVIRONMENT"); env != "" {
		config.Service.Environment = env
	}
	if prefix := os.Getenv("MW_SERVICE_PREFIX"); prefix != "" {
		config.Service.NamePrefix = prefix
	}
	if suffix := os.Getenv("MW_SERVICE_SUFFIX"); suffix != "" {
		config.Service.NameSuffix = suffix
	}

	// Host configuration
	if hostname := os.Getenv("MW_HOSTNAME"); hostname != "" {
		config.Host.Hostname = hostname
	}

	// Logging configuration
	if logLevel := os.Getenv("MW_LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}

	return nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate required fields
	if config.Middleware.APIKey == "" {
		return fmt.Errorf("middleware.api_key is required")
	}

	if config.Middleware.Target == "" {
		return fmt.Errorf("middleware.target is required")
	}

	// Validate log level
	validLogLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	if !contains(validLogLevels, config.Logging.Level) {
		return fmt.Errorf("invalid log level: %s (valid: %v)", config.Logging.Level, validLogLevels)
	}

	// Validate timeout
	if config.Advanced.Timeout <= 0 {
		return fmt.Errorf("advanced.timeout must be positive")
	}

	// Validate max concurrency
	if config.Advanced.MaxConcurrency <= 0 {
		return fmt.Errorf("advanced.max_concurrency must be positive")
	}

	return nil
}

// SaveConfigTemplate saves a configuration template to the specified path
func SaveConfigTemplate(path string) error {
	config := DefaultConfig()

	// Set placeholder values for required fields
	config.Middleware.APIKey = "your-api-key-here"

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config template: %w", err)
	}

	// Add comments and structure to the template
	template := `# MW Injector Configuration File
# Copy this file to one of the following locations:
#   - ./mw-injector.yaml (current directory)
#   - /etc/middleware/mw-injector.yaml (system-wide)
#   - ~/.mw-injector.yaml (user-specific)
#
# Or set MW_INJECTOR_CONFIG environment variable to specify custom path

` + string(data)

	if err := os.WriteFile(path, []byte(template), 0o644); err != nil {
		return fmt.Errorf("failed to write config template: %w", err)
	}

	return nil
}

// GetConfigSearchPaths returns the list of configuration file search paths
func GetConfigSearchPaths() []string {
	return []string{
		"./mw-injector.yaml",
		"./mw-injector.yml",
		"/etc/middleware/mw-injector.yaml",
		"/etc/middleware/mw-injector.yml",
		os.ExpandEnv("$HOME/.mw-injector.yaml"),
		os.ExpandEnv("$HOME/.mw-injector.yml"),
	}
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// IsConfigFilePresent checks if any configuration file exists
func IsConfigFilePresent() bool {
	for _, path := range GetConfigSearchPaths() {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// GetActiveConfigPath returns the path of the configuration file that would be used
func GetActiveConfigPath() string {
	// Check environment variable first
	if envConfigPath := os.Getenv("MW_INJECTOR_CONFIG"); envConfigPath != "" {
		if _, err := os.Stat(envConfigPath); err == nil {
			return envConfigPath
		}
	}

	// Check standard paths
	for _, path := range GetConfigSearchPaths() {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
