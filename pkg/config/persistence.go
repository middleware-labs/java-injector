package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultConfigDir is the default directory for configuration files
	DefaultConfigDir = "/etc/middleware/services"

	// FallbackConfigDir is used when /etc is not writable
	FallbackConfigDir = "$HOME/.config/middleware/services"
)

// ConfigPersistence handles saving and loading configurations
type ConfigPersistence struct {
	configDir string
}

// NewConfigPersistence creates a new persistence handler
func NewConfigPersistence() *ConfigPersistence {
	return &ConfigPersistence{
		configDir: getConfigDir(),
	}
}

// getConfigDir determines the appropriate config directory
func getConfigDir() string {
	// Try default location first
	if isWritable(DefaultConfigDir) {
		return DefaultConfigDir
	}

	// Fall back to user's home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userConfigDir := filepath.Join(homeDir, ".config", "middleware", "services")
		return userConfigDir
	}

	// Last resort: current directory
	return "./middleware-configs"
}

// isWritable checks if a directory is writable
func isWritable(path string) bool {
	// Try to create the directory if it doesn't exist
	if err := os.MkdirAll(path, 0o755); err != nil {
		return false
	}

	// Try to create a test file
	testFile := filepath.Join(path, ".write_test")
	if err := ioutil.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		return false
	}

	// Clean up test file
	os.Remove(testFile)
	return true
}

// Save saves a configuration to disk
func (cp *ConfigPersistence) Save(config *ProcessConfiguration) error {
	// Ensure config directory exists
	if err := os.MkdirAll(cp.configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate filename - use .json instead
	filename := sanitizeFilename(config.ServiceName) + ".json"
	filepath := filepath.Join(cp.configDir, filename)

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(filepath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Load loads a configuration from disk
func (cp *ConfigPersistence) Load(serviceName string) (*ProcessConfiguration, error) {
	filename := sanitizeFilename(serviceName) + ".json" // Changed to .json
	filepath := filepath.Join(cp.configDir, filename)

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration not found for service: %s", serviceName)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ProcessConfiguration
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// List returns all saved configurations
func (cp *ConfigPersistence) List() ([]*ProcessConfiguration, error) {
	files, err := ioutil.ReadDir(cp.configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ProcessConfiguration{}, nil
		}
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var configs []*ProcessConfiguration

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" { // Changed to .json
			continue
		}

		fullPath := filepath.Join(cp.configDir, file.Name())
		data, err := ioutil.ReadFile(fullPath)
		if err != nil {
			continue
		}

		var config ProcessConfiguration
		if err := json.Unmarshal(data, &config); err != nil {
			continue
		}

		configs = append(configs, &config)
	}

	return configs, nil
}

// Delete removes a configuration file
func (cp *ConfigPersistence) Delete(serviceName string) error {
	filename := sanitizeFilename(serviceName) + ".yaml"
	filepath := filepath.Join(cp.configDir, filename)

	if err := os.Remove(filepath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("configuration not found: %s", serviceName)
		}
		return fmt.Errorf("failed to delete config: %w", err)
	}

	return nil
}

// Exists checks if a configuration exists for a service
func (cp *ConfigPersistence) Exists(serviceName string) bool {
	filename := sanitizeFilename(serviceName) + ".yaml"
	filepath := filepath.Join(cp.configDir, filename)

	_, err := os.Stat(filepath)
	return err == nil
}

// Export exports configuration to JSON format
func (cp *ConfigPersistence) Export(config *ProcessConfiguration, outputPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}

	if err := ioutil.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

// Import imports configuration from JSON format
func (cp *ConfigPersistence) Import(inputPath string) (*ProcessConfiguration, error) {
	data, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read import file: %w", err)
	}

	var config ProcessConfiguration
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &config, nil
}

// GetConfigDir returns the current config directory
func (cp *ConfigPersistence) GetConfigDir() string {
	return cp.configDir
}

// sanitizeFilename removes unsafe characters from filenames
func sanitizeFilename(name string) string {
	// Replace unsafe characters with hyphens
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := name

	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "-")
	}

	// Limit length
	if len(result) > 200 {
		result = result[:200]
	}

	return result
}
