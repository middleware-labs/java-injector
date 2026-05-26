package discovery

import "testing"

func TestNodeExtractServiceName(t *testing.T) {
	h := &NodeHandler{}

	tests := []struct {
		name     string
		proc     *Process
		cmdArgs  []string
		wantName string
	}{
		// Priority 2 — CLI flags (--name=, --service=, SERVICE_NAME=)
		// (Priority 1 is systemd unit which requires /proc — tested separately)
		{
			name: "name flag in cmdArgs",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailEntryPoint: "server.js"},
			},
			cmdArgs:  []string{"node", "--name=billing-api", "server.js"},
			wantName: "billing-api",
		},
		{
			name: "service flag in cmdArgs",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailEntryPoint: "server.js"},
			},
			cmdArgs:  []string{"node", "--service=gateway", "server.js"},
			wantName: "gateway",
		},
		{
			name: "SERVICE_NAME env-style arg",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailEntryPoint: "server.js"},
			},
			cmdArgs:  []string{"node", "SERVICE_NAME=payments", "server.js"},
			wantName: "payments",
		},
		{
			name: "NODE_ENV production filtered",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailEntryPoint: "server.js", DetailPackageName: "my-api"},
			},
			cmdArgs:  []string{"node", "NODE_ENV=production", "server.js"},
			wantName: "my-api",
		},
		{
			name: "NODE_ENV development filtered",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailEntryPoint: "server.js", DetailPackageName: "my-api"},
			},
			cmdArgs:  []string{"node", "NODE_ENV=development", "server.js"},
			wantName: "my-api",
		},

		// Priority 3 — package.json name
		{
			name: "package name used",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailPackageName: "billing-service", DetailEntryPoint: "index.js"},
			},
			cmdArgs:  []string{"node", "index.js"},
			wantName: "billing-service",
		},
		{
			name: "scoped package name cleaned",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailPackageName: "@myorg/api-service", DetailEntryPoint: "index.js"},
			},
			cmdArgs:  []string{"node", "index.js"},
			wantName: "myorgapi-service",
		},
		{
			name: "generic package name falls through",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailPackageName: "app", DetailEntryPoint: "billing.js"},
			},
			cmdArgs:  []string{"node", "billing.js"},
			wantName: "billing",
		},
		{
			name: "unknown package name falls through",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailPackageName: "unknown", DetailEntryPoint: "billing.js"},
			},
			cmdArgs:  []string{"node", "billing.js"},
			wantName: "billing",
		},

		// Priority 4 — Entry point filename
		{
			name: "script filename",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailEntryPoint: "billing.js"},
			},
			cmdArgs:  []string{"node", "billing.js"},
			wantName: "billing",
		},
		{
			name: "extensionless script",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailEntryPoint: "codegraph"},
			},
			cmdArgs:  []string{"node", "codegraph"},
			wantName: "codegraph",
		},
		{
			name: "mjs extension stripped",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailEntryPoint: "worker.mjs"},
			},
			cmdArgs:  []string{"node", "worker.mjs"},
			wantName: "worker",
		},
		{
			name: "generic script index.js falls through to workdir",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{
					DetailEntryPoint:       "index.js",
					DetailWorkingDirectory: "/home/deploy/billing-api/src",
				},
			},
			cmdArgs:  []string{"node", "index.js"},
			wantName: "deploy-billing-api",
		},
		{
			name: "generic script app.js falls through to workdir",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{
					DetailEntryPoint:       "app.js",
					DetailWorkingDirectory: "/home/deploy/gateway/dist",
				},
			},
			cmdArgs:  []string{"node", "app.js"},
			wantName: "deploy-gateway",
		},
		{
			name: "generic script server.js falls through",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{
					DetailEntryPoint:       "server.js",
					DetailWorkingDirectory: "/srv/payments/backend",
				},
			},
			cmdArgs:  []string{"node", "server.js"},
			wantName: "payments-backend",
		},

		// Priority 5 — Working directory
		{
			name: "working directory used",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailWorkingDirectory: "/home/deploy/billing-api/backend"},
			},
			cmdArgs:  []string{"node"},
			wantName: "billing-api-backend",
		},

		// Priority 6 — Fallback
		{
			name: "nothing set falls to default",
			proc: &Process{
				PID:     99999,
				Details: make(map[string]any),
			},
			cmdArgs:  []string{"node"},
			wantName: "node-service",
		},
		{
			name: "all generic falls to default",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailEntryPoint: "index.js", DetailWorkingDirectory: "/opt/app"},
			},
			cmdArgs:  []string{"node", "index.js"},
			wantName: "node-service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h.extractServiceName(tt.proc, tt.cmdArgs)
			if tt.proc.ServiceName != tt.wantName {
				t.Errorf("ServiceName = %q, want %q", tt.proc.ServiceName, tt.wantName)
			}
		})
	}
}

func TestExtractNodeServiceNameFromCmdArgs(t *testing.T) {
	tests := []struct {
		name    string
		cmdArgs []string
		want    string
	}{
		{"name flag", []string{"node", "--name=billing-api", "server.js"}, "billing-api"},
		{"service flag", []string{"node", "--service=gateway", "server.js"}, "gateway"},
		{"SERVICE_NAME env", []string{"node", "SERVICE_NAME=payments", "server.js"}, "payments"},
		{"NODE_ENV production rejected", []string{"node", "NODE_ENV=production", "server.js"}, ""},
		{"NODE_ENV development rejected", []string{"node", "NODE_ENV=development", "server.js"}, ""},
		{"NODE_ENV staging accepted", []string{"node", "NODE_ENV=staging", "server.js"}, "staging"},
		{"quoted value", []string{"node", `--name="my-app"`, "server.js"}, "my-app"},
		{"single quoted value", []string{"node", "--name='my-app'", "server.js"}, "my-app"},
		{"no match", []string{"node", "server.js"}, ""},
		{"empty args", []string{}, ""},
		{"generic name cleaned away", []string{"node", "--name=app"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNodeServiceNameFromCmdArgs(tt.cmdArgs)
			if got != tt.want {
				t.Errorf("extractNodeServiceNameFromCmdArgs(%v) = %q, want %q", tt.cmdArgs, got, tt.want)
			}
		})
	}
}

func TestExtractNameFromNodeScript(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Non-generic scripts
		{"billing.js", "billing.js", "billing"},
		{"worker.mjs", "worker.mjs", "worker"},
		{"codegraph extensionless", "codegraph", "codegraph"},
		{"api-gateway.ts", "api-gateway.ts", "api-gateway"},

		// Generic scripts rejected
		{"index.js generic", "index.js", ""},
		{"app.js generic", "app.js", ""},
		{"main.js generic", "main.js", ""},
		{"server.js generic", "server.js", ""},
		{"start.js generic", "start.js", ""},
		{"run.js generic", "run.js", ""},
		{"node generic", "node", ""},
		{"npm generic", "npm", ""},
		{"yarn generic", "yarn", ""},
		{"nodemon generic", "nodemon", ""},
		{"pm2 generic", "pm2", ""},
		{"forever generic", "forever", ""},

		// Empty
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNameFromNodeScript(tt.input)
			if got != tt.want {
				t.Errorf("extractNameFromNodeScript(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
