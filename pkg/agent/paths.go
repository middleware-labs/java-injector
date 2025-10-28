package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateAgentPath checks if the agent path is valid (basic validation only)
func ValidateAgentPath(agentPath string) error {
	// Check if file exists
	if _, err := os.Stat(agentPath); err != nil {
		return fmt.Errorf("agent file does not exist: %s", agentPath)
	}

	// Check if it's a JAR file
	if !strings.HasSuffix(agentPath, ".jar") {
		return fmt.Errorf("agent file must be a .jar file: %s", agentPath)
	}

	return nil
}

// ValidateAgentPathForUser checks if the agent path is valid and accessible by user
func ValidateAgentPathForUser(agentPath string, processOwner string) error {
	// First do basic validation
	if err := ValidateAgentPath(agentPath); err != nil {
		return err
	}

	// Check if the service user can read the file
	if err := CheckAccessibleByUser(agentPath, processOwner); err != nil {
		return fmt.Errorf("user '%s' cannot read agent file: %s", processOwner, agentPath)
	}

	return nil
}

// GetAgentInfo extracts information about an agent file
func GetAgentInfo(agentPath string) (*Agent, error) {
	info, err := os.Stat(agentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent info: %w", err)
	}

	return &Agent{
		Path:    agentPath,
		Name:    filepath.Base(agentPath),
		Version: "unknown", // TODO: Extract version from JAR manifest
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}
