package javanaming

import (
	"testing"

	"github.com/middleware-labs/java-injector/pkg/discovery"
)

func TestCleanServiceName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"basic", "my_app", "my-app"},
		{"uppercase", "MyService", "myservice"},
		{"special chars", "app@v2.1", "appv21"},
		{"generic rejected", "app", ""},
		{"generic server", "service", ""},
		{"generic main", "main", ""},
		{"hyphens collapsed", "a--b", "a-b"},
		{"leading trailing hyphens", "---billing---", "billing"},
		{"empty", "", ""},
		{"already clean", "billing-api", "billing-api"},
		{"complex version stripped", "billing-api-v2", "billing-api-v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanServiceName(tt.input)
			if got != tt.want {
				t.Errorf("CleanServiceName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanJarName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"semver", "app-1.0.0.jar", ""},
		{"semver with SNAPSHOT", "billing-1.0.0-SNAPSHOT.jar", "billing"},
		{"underscore semver", "my-service_1.2.3.jar", "my-service"},
		{"BUILD suffix", "app-BUILD-42.jar", ""},
		{"no version", "myapp.jar", "myapp"},
		{"no .jar extension", "plain", "plain"},
		{"empty", "", ""},
		{"spring boot complex", "spring-boot-starter-web-3.2.1-SNAPSHOT.jar", "spring-boot-starter-web"},
		{"SNAPSHOT only", "billing-SNAPSHOT.jar", "billing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanJarName(tt.input)
			if got != tt.want {
				t.Errorf("CleanJarName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanTomcatInstance(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"apache prefix and version stripped", "apache-tomcat-9.0.50", "tomcat"},
		{"path with apache", "/opt/apache-tomcat-10.1.2/bin", "bin"},
		{"custom instance", "my-webapp-instance", "my-webapp-instance"},
		{"just tomcat passes", "tomcat", "tomcat"},
		{"path base used", "/opt/billing/catalina", "catalina"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanTomcatInstance(tt.input)
			if got != tt.want {
				t.Errorf("CleanTomcatInstance(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsGenericName(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"java", true}, {"app", true}, {"application", true},
		{"service", true}, {"server", true}, {"main", true},
		{"demo", true}, {"test", true}, {"example", true},
		{"sample", true}, {"hello", true}, {"world", true},

		// Case insensitive
		{"Java", true}, {"APP", true}, {"Service", true},

		// Not generic
		{"billing-api", false}, {"user-service", false},
		{"apps", false}, {"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsGenericName(tt.input)
			if got != tt.want {
				t.Errorf("IsGenericName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateForStandard(t *testing.T) {
	tests := []struct {
		name string
		proc *discovery.Process
		want string
	}{
		{
			name: "JAR file used",
			proc: &discovery.Process{
				PID:     1234,
				Details: map[string]any{discovery.DetailJarFile: "billing-api-1.2.3.jar"},
			},
			want: "billing-api",
		},
		{
			name: "generic JAR falls through to ServiceName",
			proc: &discovery.Process{
				PID:         1234,
				ServiceName: "gateway",
				Details:     map[string]any{discovery.DetailJarFile: "app-1.0.0.jar"},
			},
			want: "gateway",
		},
		{
			name: "no JAR uses ServiceName",
			proc: &discovery.Process{
				PID:         1234,
				ServiceName: "billing",
				Details:     make(map[string]any),
			},
			want: "billing",
		},
		{
			name: "java-service ServiceName is skipped",
			proc: &discovery.Process{
				PID:         1234,
				ServiceName: "java-service",
				Details:     make(map[string]any),
			},
			want: "java-app-1234",
		},
		{
			name: "nothing set falls to PID",
			proc: &discovery.Process{
				PID:     5678,
				Details: make(map[string]any),
			},
			want: "java-app-5678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateForStandard(tt.proc)
			if got != tt.want {
				t.Errorf("GenerateForStandard() = %q, want %q", got, tt.want)
			}
		})
	}
}
