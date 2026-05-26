package discovery

import "testing"

func TestCleanName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Basic sanitization
		{"lowercase", "MyService", "myservice"},
		{"underscore to dash", "my_cool_app", "my-cool-app"},
		{"special chars removed", "app@v2.1", "appv21"},
		{"already clean", "billing-api", "billing-api"},

		// Dash handling (---app → strips to "app" which is generic → "")
		{"leading dashes around generic", "---app", ""},
		{"trailing dashes around generic", "app---", ""},
		{"leading dashes non-generic", "---billing", "billing"},
		{"trailing dashes non-generic", "billing---", "billing"},
		{"multi dashes collapsed", "a--b---c", "a-b-c"},

		// Generic rejection
		{"generic java", "java", ""},
		{"generic app", "app", ""},
		{"generic application", "application", ""},
		{"generic service", "service", ""},
		{"generic server", "server", ""},
		{"generic main", "main", ""},
		{"generic demo", "demo", ""},
		{"generic test", "test", ""},
		{"generic example", "example", ""},
		{"generic sample", "sample", ""},
		{"generic hello", "hello", ""},
		{"generic world", "world", ""},

		// Generic is case-insensitive (lowered before check)
		{"generic uppercase APP", "APP", ""},
		{"generic mixed Java", "Java", ""},

		// Non-generic passes
		{"billing-api passes", "billing-api", "billing-api"},
		{"user-service passes", "user-service", "user-service"},
		{"apps is not generic", "apps", "apps"},
		{"servers is not generic", "servers", "servers"},

		// Empty / edge
		{"empty string", "", ""},
		{"only special chars", "@#$%", ""},
		{"only dashes", "---", ""},

		// Realistic names
		{"spring style", "billing-api-v2", "billing-api-v2"},
		{"dotted version stripped", "my.service.v1", "myservicev1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanName(tt.input)
			if got != tt.want {
				t.Errorf("cleanName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestServiceNameFromWorkDir(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Standard 2-segment extraction
		{"two meaningful segments", "/home/user/billing-api/backend", "billing-api-backend"},
		{"project and subdir", "/srv/deploy/billing/api", "billing-api"},

		// Single meaningful segment
		{"one meaningful segment", "/opt/billing", "billing"},
		{"deep but one meaningful", "/home/usr/billing", "billing"},

		// Generic dirs filtered
		{"all generic", "/opt/app", ""},
		{"home generic", "/home", ""},
		{"root path", "/", ""},
		{"empty string", "", ""},
		{"tmp generic", "/tmp/app", ""},
		{"var www filtered", "/var/app", ""},

		// Last 2 meaningful picked
		{"deep path picks last 2", "/a/b/c/payments/api", "payments-api"},

		// Generic in the middle skipped
		{"generic middle", "/data/src/billing/lib/handlers", "billing-handlers"},
		{"build is generic", "/home/user/billing/build", "user-billing"},

		// Real-world paths
		{"node project", "/home/deploy/browse-bay/backend", "browse-bay-backend"},
		{"pm2 path", "/usr/lib/node-modules/pm2", "pm2"},

		// cleanName applied to result
		{"underscores converted", "/srv/my_project/my_app", "my-project-my-app"},

		// Result is generic after join → empty
		{"result becomes generic", "/opt/app/main", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serviceNameFromWorkDir(tt.input)
			if got != tt.want {
				t.Errorf("serviceNameFromWorkDir(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
