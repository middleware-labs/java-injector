package discovery

import (
	"testing"
)

func TestJavaExtractServiceName(t *testing.T) {
	h := &JavaHandler{}

	tests := []struct {
		name     string
		proc     *Process
		cmdArgs  []string
		wantName string
	}{
		{
			name: "container name wins over everything",
			proc: &Process{
				PID:           99999,
				Details:       map[string]any{DetailJarFile: "payments-1.0.0.jar"},
				ContainerInfo: &ContainerInfo{IsContainer: true, ContainerName: "billing-api"},
			},
			cmdArgs:  []string{"java", "-Dspring.application.name=gateway", "-jar", "payments.jar"},
			wantName: "billing-api",
		},
		{
			name: "container present but no name falls through",
			proc: &Process{
				PID:           99999,
				Details:       map[string]any{DetailJarFile: "payments-1.0.0.jar"},
				ContainerInfo: &ContainerInfo{IsContainer: true, ContainerName: ""},
			},
			wantName: "payments",
		},
		{
			name: "not in container falls through",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailJarFile: "payments-1.0.0.jar"},
			},
			wantName: "payments",
		},
		{
			name: "system property wins over JAR",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailJarFile: "generic-app-1.0.0.jar"},
			},
			cmdArgs:  []string{"java", "-Dspring.application.name=gateway", "-jar", "generic-app.jar"},
			wantName: "gateway",
		},
		{
			name: "JAR file name",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailJarFile: "billing-api-1.2.3.jar"},
			},
			wantName: "billing-api",
		},
		{
			name: "generic JAR falls through to main class",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailJarFile: "app.jar", DetailMainClass: "com.example.PaymentService"},
			},
			wantName: "payment",
		},
		{
			name: "main class only",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailMainClass: "com.example.OrderProcessing"},
			},
			wantName: "order-processing",
		},
		{
			name: "generic main class falls through to JAR path",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailMainClass: "com.example.Main", DetailJarPath: "/srv/billing/target/app.jar"},
			},
			wantName: "billing",
		},
		{
			name: "JAR path directory only",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailJarPath: "/srv/billing-service/target/app.jar"},
			},
			wantName: "billing-service",
		},
		{
			name:     "nothing set falls to default",
			proc:     &Process{PID: 99999, Details: make(map[string]any)},
			wantName: "java-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h.extractServiceName(tt.proc, tt.cmdArgs)
			if tt.proc.ServiceName != tt.wantName {
				t.Errorf("Serive name = %q, want %q", tt.proc.ServiceName, tt.wantName)
			}
		})
	}
}

func TestExtractServiceNameFromSystemProperties(t *testing.T) {

	tests := []struct {
		name    string
		cmdArgs []string
		want    string
	}{
		// Each of the 7 prefixes
		{"otel.service.name", []string{"java", "-Dotel.service.name=billing", "-jar", "app.jar"}, "billing"},
		{"service.name", []string{"java", "-Dservice.name=billing", "-jar", "app.jar"}, "billing"},
		{"spring.application.name", []string{"java", "-Dspring.application.name=billing", "-jar", "app.jar"}, "billing"},
		{"application.name", []string{"java", "-Dapplication.name=billing", "-jar", "app.jar"}, "billing"},
		{"mw.service.name", []string{"java", "-Dmw.service.name=billing", "-jar", "app.jar"}, "billing"},
		{"OTEL_SERVICE_NAME uppercase", []string{"java", "-DOTEL_SERVICE_NAME=billing", "-jar", "app.jar"}, "billing"},
		{"SERVICE_NAME uppercase", []string{"java", "-DSERVICE_NAME=billing", "-jar", "app.jar"}, "billing"},

		// Priority: first match in cmdArgs wins
		{"first match wins", []string{"java", "-Dotel.service.name=first", "-Dspring.application.name=second"}, "first"},

		// Quoted values stripped
		{"single quotes stripped", []string{"java", "-Dspring.application.name='my-app'"}, "my-app"},
		{"double quotes stripped", []string{"java", `-Dspring.application.name="my-app"`}, "my-app"},

		// Empty value
		{"empty value after equals", []string{"java", "-Dspring.application.name="}, ""},

		// No match
		{"no matching property", []string{"java", "-jar", "app.jar"}, ""},
		{"empty args", []string{}, ""},

		// Generic name cleaned to empty
		{"generic name rejected", []string{"java", "-Dspring.application.name=app"}, ""},

		// Underscore converted
		{"underscore converted", []string{"java", "-Dspring.application.name=my_cool_service"}, "my-cool-service"},

		// Property buried among other JVM args
		{"buried among other args", []string{"java", "-XX:+UseG1GC", "-Xmx512m", "-Dspring.application.name=billing", "-jar", "app.jar"}, "billing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFromJavaSystemProperties(tt.cmdArgs)
			if got != tt.want {
				t.Errorf("extractFromJavaSystemProperties() = %q, want = %q", got, tt.want)
			}
		})
	}
}

func TestExtractNameFromMainClass(t *testing.T) {
	tests := []struct {
		name      string
		mainClass string
		want      string
	}{
		// Suffix stripping
		{"Service suffix", "com.example.BillingService", "billing"},
		{"Application suffix", "com.example.UserApplication", "user"},
		{"App suffix", "com.example.PaymentApp", "payment"},
		{"Server suffix", "com.example.GatewayServer", "gateway"},
		{"Main suffix", "com.example.Main", ""},
		{"Launcher suffix", "com.example.TaskLauncher", "task"},
		{"Bootstrap suffix", "com.example.Bootstrap", ""},

		{"Service suffix", "com.example.BillingService", "billing"},
		{"Application suffix", "com.example.UserApplication", "user"},
		{"App suffix", "com.example.PaymentApp", "payment"},
		{"Server suffix", "com.example.GatewayServer", "gateway"},
		{"Main suffix", "com.example.Main", ""},
		{"Launcher suffix", "com.example.TaskLauncher", "task"},
		{"Bootstrap suffix", "com.example.Bootstrap", ""},

		// CamelCase splitting
		{"camel case split", "com.example.OrderProcessingService", "order-processing"},
		{"multi word camel", "com.example.UserAccountApplication", "user-account"},

		// Chained suffix stripping
		{"AppLauncher both stripped", "com.example.AppLauncher", ""},
		{"ApplicationService both stripped", "com.example.ApplicationService", ""},

		// No package
		{"no package prefix", "BillingService", "billing"},

		// Spring Boot loader
		{"spring JarLauncher", "org.springframework.boot.loader.JarLauncher", "jar"},

		// Empty
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNameFromMainClass(tt.mainClass)
			if got != tt.want {
				t.Errorf("extractNameFromMainClass(%q) = %q, want %q", tt.mainClass, got, tt.want)
			}
		})
	}
}

func TestExtractNameFromDir(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Meaningful parent dir
		{"meaningful parent", "/home/user/billing-api/app.jar", "billing-api"},

		// Deeper path picks last meaningful
		{"deep path last meaningful", "/srv/deploy/billing/app.jar", "billing"},
		{"deep with lib", "/data/services/payment-gateway/lib/agent.jar", "payment-gateway"},

		// Generic parents only
		{"opt only", "/opt/app.jar", ""},
		{"tmp only", "/tmp/app.jar", ""},

		// Generic dir before file skipped
		{"target is generic", "/home/user/target/app.jar", "user"},
		{"build is generic", "/home/user/build/app.jar", "user"},
		{"java is generic", "/opt/java/app.jar", ""},
		{"jvm is generic", "/usr/lib/jvm/app.jar", ""},

		// Empty
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNameFromDir(tt.input)
			if got != tt.want {
				t.Errorf("extractNameFromDir(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripJarVersionExpanded(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Patterns the existing test misses:

		// Pattern 2: -major.minor (dash, 2-part)
		{"dash 2-part version", "app-1.0.jar", "app"},
		{"dash 2-part with qualifier", "service-2.1.RELEASE.jar", "service"},

		// Pattern 4: _major.minor (underscore, 2-part)
		{"underscore 2-part version", "app_1.0.jar", "app"},

		// Pattern 6: _SNAPSHOT
		{"underscore SNAPSHOT", "app_SNAPSHOT.jar", "app"},

		// Pattern 8: _BUILD_\d+
		{"underscore BUILD", "app_BUILD_42.jar", "app"},

		// Path prefix — filepath.Base strips it
		{"path prefix stripped", "/opt/jars/billing-1.0.0.jar", "billing"},

		// Combo: semver catches before SNAPSHOT
		{"semver plus SNAPSHOT", "billing-1.0.0-SNAPSHOT.jar", "billing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripJarVersion(tt.input)
			if got != tt.want {
				t.Errorf("stripJarVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsGenericJava(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// All generics
		{"bin", true}, {"lib", true}, {"src", true}, {"target", true},
		{"java", true}, {"jre", true}, {"jdk", true}, {"app", true},
		{"main", true}, {"server", true}, {"tomcat", true},
		{"", true}, {".", true},

		// Case insensitive
		{"Java", true}, {"BIN", true}, {"Tomcat", true}, {"TARGET", true},

		// Not generic
		{"billing", false}, {"myapp", false}, {"gateway", false},
		{"apps", false}, {"servers", false}, {"libraries", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isGenericJava(tt.input)
			if got != tt.want {
				t.Errorf("isGenericJava(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
