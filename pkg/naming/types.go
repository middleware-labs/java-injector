package naming

// ServiceType represents the type of Java service for naming purposes
type ServiceType int

const (
	ServiceTypeStandard ServiceType = iota
	ServiceTypeTomcat
	ServiceTypeSpringBoot
	ServiceTypeGeneric
)

// ServiceNameOptions holds options for service name generation
type ServiceNameOptions struct {
	PreferredName string
	ServiceType   ServiceType
	AllowGeneric  bool
}

// String returns a string representation of ServiceType
func (st ServiceType) String() string {
	switch st {
	case ServiceTypeStandard:
		return "standard"
	case ServiceTypeTomcat:
		return "tomcat"
	case ServiceTypeSpringBoot:
		return "springboot"
	case ServiceTypeGeneric:
		return "generic"
	default:
		return "unknown"
	}
}
