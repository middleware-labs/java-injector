package agent

import (
	"os"
	"time"
)

// Agent represents information about a Java agent
type Agent struct {
	Path    string
	Name    string
	Version string
	Size    int64
	ModTime time.Time
}

// InstallationConfig holds configuration for agent installation
type InstallationConfig struct {
	SourcePath    string
	TargetPath    string
	RequiredPerms os.FileMode
	OwnerUID      int
	OwnerGID      int
}

// ValidationResult holds the result of agent validation
type ValidationResult struct {
	Exists      bool
	Readable    bool
	Permissions os.FileMode
	Error       error
}
