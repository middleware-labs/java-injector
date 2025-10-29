package commands

import (
	"fmt"
	"os"

	"github.com/middleware-labs/java-injector/pkg/cli/types"
	"github.com/middleware-labs/java-injector/pkg/config"
)

// ConfigInitCommand creates a configuration file template
type ConfigInitCommand struct {
	config *types.CommandConfig
	path   string
}

func NewConfigInitCommand(config *types.CommandConfig) *ConfigInitCommand {
	return &ConfigInitCommand{config: config}
}

func (c *ConfigInitCommand) SetArg(arg string) {
	c.path = arg
}

func (c *ConfigInitCommand) Execute() error {
	// Determine output path
	outputPath := c.path
	if outputPath == "" {
		outputPath = "./mw-injector.yaml"
	}

	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("❌ Configuration file already exists at: %s\n   Use --force to overwrite or specify a different path", outputPath)
	}

	// Create configuration template
	if err := config.SaveConfigTemplate(outputPath); err != nil {
		return fmt.Errorf("❌ Failed to create configuration template: %w", err)
	}

	fmt.Printf("✅ Configuration template created: %s\n\n", outputPath)
	fmt.Println("📝 Next steps:")
	fmt.Println("   1. Edit the configuration file and set your API key")
	fmt.Println("   2. Review and customize other settings as needed")
	fmt.Println("   3. Run: mw-injector auto-instrument")
	fmt.Println()
	fmt.Printf("💡 Tip: You can also set MW_API_KEY environment variable instead of editing the file\n")

	return nil
}

func (c *ConfigInitCommand) GetDescription() string {
	return "Create a configuration file template"
}

// ConfigValidateCommand validates the current configuration
type ConfigValidateCommand struct {
	config *types.CommandConfig
}

func NewConfigValidateCommand(config *types.CommandConfig) *ConfigValidateCommand {
	return &ConfigValidateCommand{config: config}
}

func (c *ConfigValidateCommand) Execute() error {
	fmt.Println("🔍 Validating configuration...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("❌ Configuration validation failed: %w", err)
	}

	// Show active configuration path
	if configPath := config.GetActiveConfigPath(); configPath != "" {
		fmt.Printf("📋 Using configuration file: %s\n", configPath)
	} else {
		fmt.Println("📋 Using default configuration (no config file found)")
	}

	// Display configuration summary
	fmt.Printf("\n✅ Configuration is valid!\n\n")
	fmt.Printf("📊 Configuration Summary:\n")
	fmt.Printf("   API Key: %s*** (length: %d)\n", cfg.Middleware.APIKey[:min(8, len(cfg.Middleware.APIKey))], len(cfg.Middleware.APIKey))
	fmt.Printf("   Target: %s\n", cfg.Middleware.Target)
	fmt.Printf("   Agent Path: %s\n", cfg.Agent.Path)
	fmt.Printf("   Environment: %s\n", cfg.Service.Environment)
	fmt.Printf("   Auto Confirm: %t\n", cfg.Instrumentation.AutoConfirm)
	fmt.Printf("   Restart Services: %t\n", cfg.Instrumentation.RestartServices)
	fmt.Printf("   Log Level: %s\n", cfg.Logging.Level)

	if len(cfg.Service.Tags) > 0 {
		fmt.Printf("   Tags: %v\n", cfg.Service.Tags)
	}

	return nil
}

func (c *ConfigValidateCommand) GetDescription() string {
	return "Validate the current configuration"
}

// ConfigShowCommand displays the current configuration
type ConfigShowCommand struct {
	config *types.CommandConfig
}

func NewConfigShowCommand(config *types.CommandConfig) *ConfigShowCommand {
	return &ConfigShowCommand{config: config}
}

func (c *ConfigShowCommand) Execute() error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("❌ Failed to load configuration: %w", err)
	}

	// Show configuration file locations
	fmt.Println("📁 Configuration File Search Paths:")
	for i, path := range config.GetConfigSearchPaths() {
		exists := ""
		if _, err := os.Stat(path); err == nil {
			exists = " ✅"
		}
		fmt.Printf("   %d. %s%s\n", i+1, path, exists)
	}

	// Show active configuration
	if activeConfig := config.GetActiveConfigPath(); activeConfig != "" {
		fmt.Printf("\n📋 Active Configuration: %s\n", activeConfig)
	} else {
		fmt.Println("\n📋 Using Default Configuration (no config file found)")
	}

	// Show environment variable overrides
	fmt.Println("\n🌍 Environment Variable Overrides:")
	envVars := []struct {
		name, value string
	}{
		{"MW_API_KEY", os.Getenv("MW_API_KEY")},
		{"MW_TARGET", os.Getenv("MW_TARGET")},
		{"MW_AGENT_PATH", os.Getenv("MW_AGENT_PATH")},
		{"MW_ENVIRONMENT", os.Getenv("MW_ENVIRONMENT")},
		{"MW_LOG_LEVEL", os.Getenv("MW_LOG_LEVEL")},
		{"MW_INJECTOR_CONFIG", os.Getenv("MW_INJECTOR_CONFIG")},
	}

	hasOverrides := false
	for _, env := range envVars {
		if env.value != "" {
			if !hasOverrides {
				hasOverrides = true
			}
			if env.name == "MW_API_KEY" {
				fmt.Printf("   %s=%s***\n", env.name, env.value[:min(8, len(env.value))])
			} else {
				fmt.Printf("   %s=%s\n", env.name, env.value)
			}
		}
	}

	if !hasOverrides {
		fmt.Println("   (none)")
	}

	// Display full configuration (sanitized)
	fmt.Printf("\n📊 Final Configuration:\n")
	fmt.Printf("   Middleware:\n")
	fmt.Printf("     API Key: %s*** (length: %d)\n", cfg.Middleware.APIKey[:min(8, len(cfg.Middleware.APIKey))], len(cfg.Middleware.APIKey))
	fmt.Printf("     Target: %s\n", cfg.Middleware.Target)
	fmt.Printf("   Agent:\n")
	fmt.Printf("     Path: %s\n", cfg.Agent.Path)
	fmt.Printf("     Auto Download: %t\n", cfg.Agent.AutoDownload)
	fmt.Printf("   Service:\n")
	fmt.Printf("     Environment: %s\n", cfg.Service.Environment)
	if cfg.Service.NamePrefix != "" {
		fmt.Printf("     Name Prefix: %s\n", cfg.Service.NamePrefix)
	}
	if cfg.Service.NameSuffix != "" {
		fmt.Printf("     Name Suffix: %s\n", cfg.Service.NameSuffix)
	}
	if len(cfg.Service.Tags) > 0 {
		fmt.Printf("     Tags: %v\n", cfg.Service.Tags)
	}
	fmt.Printf("   Host:\n")
	fmt.Printf("     Hostname: %s\n", cfg.Host.Hostname)
	fmt.Printf("     Include Hostname: %t\n", cfg.Host.IncludeHostname)
	fmt.Printf("   Instrumentation:\n")
	fmt.Printf("     Auto Confirm: %t\n", cfg.Instrumentation.AutoConfirm)
	fmt.Printf("     Skip Instrumented: %t\n", cfg.Instrumentation.SkipInstrumented)
	fmt.Printf("     Restart Services: %t\n", cfg.Instrumentation.RestartServices)
	fmt.Printf("   Logging:\n")
	fmt.Printf("     Level: %s\n", cfg.Logging.Level)
	if cfg.Logging.File != "" {
		fmt.Printf("     File: %s\n", cfg.Logging.File)
	}

	return nil
}

func (c *ConfigShowCommand) GetDescription() string {
	return "Show current configuration and file locations"
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
