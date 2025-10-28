package state

import (
	"fmt"
	"os"
	"time"
)

// DockerStateManager provides utilities for Docker state management
// This complements the docker package without replacing its state logic
type DockerStateManager struct {
	stateFile string
}

// NewDockerStateManager creates a new Docker state manager
func NewDockerStateManager() *DockerStateManager {
	return &DockerStateManager{
		stateFile: DockerStateFile,
	}
}

// LoadDockerState loads Docker instrumentation state using state utilities
func (dsm *DockerStateManager) LoadDockerState() (map[string]interface{}, error) {
	var state map[string]interface{}

	if err := LoadFromFile(dsm.stateFile, &state); err != nil {
		return nil, fmt.Errorf("failed to load Docker state: %w", err)
	}

	// Initialize empty state if file doesn't exist
	if state == nil {
		state = map[string]interface{}{
			"containers": make(map[string]interface{}),
			"updated_at": time.Now(),
			"version":    StateVersion,
		}
	}

	return state, nil
}

// SaveDockerState saves Docker instrumentation state using state utilities
func (dsm *DockerStateManager) SaveDockerState(state map[string]interface{}) error {
	// Update metadata
	state["updated_at"] = time.Now()
	state["version"] = StateVersion

	if err := SaveToFile(dsm.stateFile, state); err != nil {
		return fmt.Errorf("failed to save Docker state: %w", err)
	}

	return nil
}

// ValidateDockerState validates Docker state file structure
func (dsm *DockerStateManager) ValidateDockerState() *ValidationResult {
	requiredFields := []string{"containers", "updated_at"}
	return ValidateStateFile(dsm.stateFile, requiredFields)
}

// BackupDockerState creates a backup of the Docker state file
func (dsm *DockerStateManager) BackupDockerState() error {
	return BackupStateFile(dsm.stateFile)
}

// GetDockerStatePath returns the full path to the Docker state file
func (dsm *DockerStateManager) GetDockerStatePath() string {
	return GetStateFilePath(dsm.stateFile)
}

// DockerStateExists checks if Docker state file exists
func (dsm *DockerStateManager) DockerStateExists() bool {
	return StateFileExists(dsm.stateFile)
}

// MigrateDockerState migrates Docker state from old location to new centralized location
func (dsm *DockerStateManager) MigrateDockerState(oldPath string) error {
	// Check if old state file exists
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return nil // Nothing to migrate
	}

	// Read old state file
	oldData, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read old state file: %w", err)
	}

	// Ensure new state directory exists
	if err := EnsureStateDir(); err != nil {
		return err
	}

	// Write to new location
	newPath := dsm.GetDockerStatePath()
	if err := os.WriteFile(newPath, oldData, 0o644); err != nil {
		return fmt.Errorf("failed to write migrated state: %w", err)
	}

	// Create backup of old file before removing
	backupPath := oldPath + ".migrated.backup"
	if err := os.Rename(oldPath, backupPath); err != nil {
		fmt.Printf("Warning: Could not backup old state file: %v\n", err)
	} else {
		fmt.Printf("Migrated Docker state from %s to %s\n", oldPath, newPath)
		fmt.Printf("Old state backed up to %s\n", backupPath)
	}

	return nil
}

// Utility functions for Docker package integration

// GetDefaultDockerStateFile returns the default Docker state file name
func GetDefaultDockerStateFile() string {
	return DockerStateFile
}

// GetDockerStateDir returns the directory for Docker state files
func GetDockerStateDir() string {
	return DefaultStateDir
}
