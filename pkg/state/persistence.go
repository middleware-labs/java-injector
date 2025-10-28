package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EnsureStateDir ensures the state directory exists with proper permissions
func EnsureStateDir() error {
	if err := os.MkdirAll(DefaultStateDir, 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	return nil
}

// LoadFromFile loads JSON data from a state file
func LoadFromFile(filename string, data interface{}) error {
	path := filepath.Join(DefaultStateDir, filename)

	// Return empty data if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	fileData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read state file %s: %w", filename, err)
	}

	if err := json.Unmarshal(fileData, data); err != nil {
		return fmt.Errorf("failed to unmarshal state file %s: %w", filename, err)
	}

	return nil
}

// SaveToFile saves JSON data to a state file
func SaveToFile(filename string, data interface{}) error {
	if err := EnsureStateDir(); err != nil {
		return err
	}

	path := filepath.Join(DefaultStateDir, filename)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state data: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0o644); err != nil {
		return fmt.Errorf("failed to write state file %s: %w", filename, err)
	}

	return nil
}

// LoadHostState loads the host instrumentation state
func LoadHostState() (*HostState, error) {
	state := &HostState{
		OrphanedConfigs: []OrphanedConfig{},
		Version:         StateVersion,
	}

	if err := LoadFromFile(HostStateFile, state); err != nil {
		return nil, err
	}

	return state, nil
}

// SaveHostState saves the host instrumentation state
func SaveHostState(state *HostState) error {
	state.LastScan = time.Now()
	state.Version = StateVersion
	return SaveToFile(HostStateFile, state)
}

// ValidateStateFile validates a state file structure
func ValidateStateFile(filename string, requiredFields []string) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []string{},
		Warnings: []string{},
	}

	path := filepath.Join(DefaultStateDir, filename)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("State file %s does not exist", filename))
		return result
	}

	// Try to load as generic JSON
	var data map[string]interface{}
	if err := LoadFromFile(filename, &data); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to load state file: %v", err))
		return result
	}

	// Check required fields
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Missing required field: %s", field))
		}
	}

	return result
}

// CleanupOldStateFiles removes old or invalid state files
func CleanupOldStateFiles(maxAge time.Duration) error {
	files, err := os.ReadDir(DefaultStateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No state directory to clean
		}
		return fmt.Errorf("failed to read state directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(DefaultStateDir, file.Name())
			if err := os.Remove(path); err != nil {
				fmt.Printf("Warning: Failed to remove old state file %s: %v\n", file.Name(), err)
			} else {
				fmt.Printf("Cleaned up old state file: %s\n", file.Name())
			}
		}
	}

	return nil
}

// GetStateFilePath returns the full path to a state file
func GetStateFilePath(filename string) string {
	return filepath.Join(DefaultStateDir, filename)
}

// StateFileExists checks if a state file exists
func StateFileExists(filename string) bool {
	path := filepath.Join(DefaultStateDir, filename)
	_, err := os.Stat(path)
	return err == nil
}

// BackupStateFile creates a backup of a state file
func BackupStateFile(filename string) error {
	if !StateFileExists(filename) {
		return fmt.Errorf("state file %s does not exist", filename)
	}

	sourcePath := filepath.Join(DefaultStateDir, filename)
	backupPath := filepath.Join(DefaultStateDir, filename+".backup")

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	return nil
}
