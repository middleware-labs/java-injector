package systemd

// DropInConfig holds configuration for creating systemd drop-in files
type DropInConfig struct {
	ServiceName string
	ConfigPath  string
	IsTomcat    bool
	AgentPath   string
}

// TomcatConfig holds configuration for Tomcat services
type TomcatConfig struct {
	InstanceName string
	Pattern      string
	APIKey       string
	Target       string
	AgentPath    string
}

// StandardConfig holds configuration for standard Java services
type StandardConfig struct {
	ServiceName string
	APIKey      string
	Target      string
	AgentPath   string
}

// ServiceInfo represents information about a systemd service
type ServiceInfo struct {
	Name        string
	Path        string
	Status      string
	Environment map[string]string
}

// ConfigVars represents parsed configuration variables
type ConfigVars map[string]string
