package discovery

import "testing"

func TestPythonExtractServiceName(t *testing.T) {
	h := &PythonHandler{}

	tests := []struct {
		name     string
		proc     *Process
		cmdArgs  []string
		wantName string
	}{
		// Priority 1 — Env vars (OTEL_SERVICE_NAME, SERVICE_NAME, FLASK_APP)
		// Requires /proc — tested at integration level.
		// We can still test fallthrough by using a fake PID that has no /proc entry.

		// Priority 2 — Container name
		{
			name: "container name wins",
			proc: &Process{
				PID:           99999,
				ContainerInfo: &ContainerInfo{IsContainer: true, ContainerName: "flask-api"},
				Details:       map[string]any{DetailEntryPoint: "app.py"},
			},
			cmdArgs:  []string{"/usr/bin/python3", "app.py"},
			wantName: "flask-api",
		},
		{
			name: "container present but empty name falls through",
			proc: &Process{
				PID:           99999,
				ContainerInfo: &ContainerInfo{IsContainer: true, ContainerName: ""},
				Details:       map[string]any{DetailEntryPoint: "billing.py"},
			},
			cmdArgs:  []string{"/usr/bin/python3", "billing.py"},
			wantName: "billing",
		},
		{
			name: "not in container falls through",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailEntryPoint: "billing.py"},
			},
			cmdArgs:  []string{"/usr/bin/python3", "billing.py"},
			wantName: "billing",
		},

		// Priority 3 — VirtualEnv path analysis
		{
			name: "venv path extracts parent dir",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/home/user/billing/.venv/bin/python3", "app.py"},
				Details:     make(map[string]any),
			},
			cmdArgs:  []string{"/home/user/billing/.venv/bin/python3", "app.py"},
			wantName: "billing",
		},
		{
			name: "venv dir extracts parent",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/opt/myapp/venv/bin/python3"},
				Details:     make(map[string]any),
			},
			cmdArgs:  []string{"/opt/myapp/venv/bin/python3"},
			wantName: "myapp",
		},
		{
			name: "dotenv dir extracts parent",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/home/deploy/payments/.env/bin/python3"},
				Details:     make(map[string]any),
			},
			cmdArgs:  []string{"/home/deploy/payments/.env/bin/python3"},
			wantName: "payments",
		},
		{
			name: "venv with generic parent falls through",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/opt/app/.venv/bin/python3", "billing.py"},
				Details:     make(map[string]any),
			},
			cmdArgs:  []string{"/opt/app/.venv/bin/python3", "billing.py"},
			wantName: "billing",
		},

		// Priority 4 — Entry point / module analysis (WSGI colon notation)
		{
			name: "wsgi module:app notation",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/usr/bin/python3", "gunicorn", "myapp:create_app"},
				Details:     make(map[string]any),
			},
			cmdArgs:  []string{"/usr/bin/python3", "gunicorn", "myapp:create_app"},
			wantName: "myapp",
		},
		{
			name: "uvicorn module:app",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/usr/bin/python3", "uvicorn", "billing:app", "--host", "0.0.0.0"},
				Details:     make(map[string]any),
			},
			cmdArgs:  []string{"/usr/bin/python3", "uvicorn", "billing:app", "--host", "0.0.0.0"},
			wantName: "billing",
		},
		{
			name: "generic wsgi module falls through to entry point parent dir",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/usr/bin/python3", "gunicorn", "app:create_app"},
				Details: map[string]any{
					DetailEntryPoint:       "run.py",
					DetailWorkingDirectory: "/srv/payments/backend",
				},
			},
			cmdArgs:  []string{"/usr/bin/python3", "gunicorn", "app:create_app"},
			wantName: "backend",
		},

		// Priority 4 — .py file basename
		{
			name: "py script basename",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/usr/bin/python3", "billing.py"},
				Details:     make(map[string]any),
			},
			cmdArgs:  []string{"/usr/bin/python3", "billing.py"},
			wantName: "billing",
		},
		{
			name: "generic py script falls through to entry point parent dir",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/usr/bin/python3", "app.py"},
				Details: map[string]any{
					DetailEntryPoint:       "app.py",
					DetailWorkingDirectory: "/srv/payments/api",
				},
			},
			cmdArgs:  []string{"/usr/bin/python3", "app.py"},
			wantName: "api",
		},

		// "manage" is not generic for Python, so Level 4 catches it
		{
			name: "manage.py caught at Level 4",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/usr/bin/python3", "manage.py"},
				Details:     map[string]any{DetailEntryPoint: "manage.py", DetailWorkingDirectory: "/srv/billing"},
			},
			cmdArgs:  []string{"/usr/bin/python3", "manage.py"},
			wantName: "manage",
		},

		// Priority 5 — Script parent directory (entry point parent)
		{
			name: "entry point parent dir when script is generic",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/usr/bin/python3", "main.py"},
				Details:     map[string]any{DetailEntryPoint: "main.py", DetailWorkingDirectory: "/srv/billing"},
			},
			cmdArgs:  []string{"/usr/bin/python3", "main.py"},
			wantName: "billing",
		},

		// Priority 6 — Working directory
		{
			name: "working directory fallback",
			proc: &Process{
				PID:     99999,
				Details: map[string]any{DetailWorkingDirectory: "/home/deploy/billing-api/backend"},
			},
			cmdArgs:  []string{"/usr/bin/python3"},
			wantName: "billing-api-backend",
		},

		// Priority 7 — Default fallback
		{
			name: "nothing set falls to default",
			proc: &Process{
				PID:     99999,
				Details: make(map[string]any),
			},
			cmdArgs:  []string{"/usr/bin/python3"},
			wantName: "python-service",
		},
		{
			name: "all generic falls to default",
			proc: &Process{
				PID:         99999,
				CommandArgs: []string{"/usr/bin/python3", "app.py"},
				Details:     map[string]any{DetailEntryPoint: "app.py", DetailWorkingDirectory: "/opt/app"},
			},
			cmdArgs:  []string{"/usr/bin/python3", "app.py"},
			wantName: "python-service",
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

func TestIsGenericPython(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// All generics
		{"app", true}, {"main", true}, {"server", true}, {"index", true},
		{"python", true}, {"python3", true}, {"uvicorn", true}, {"gunicorn", true},
		{"bin", true}, {"src", true}, {"lib", true},

		// Case insensitive
		{"App", true}, {"MAIN", true}, {"Python3", true}, {"Gunicorn", true},

		// Not generic
		{"billing", false}, {"payments", false}, {"flask-api", false},
		{"myapp", false}, {"worker-service", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isGenericPython(tt.input)
			if got != tt.want {
				t.Errorf("isGenericPython(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPythonDetect(t *testing.T) {
	h := &PythonHandler{}

	tests := []struct {
		name    string
		proc    *ProcessInfo
		want    bool
	}{
		// Direct executables
		{"python", &ProcessInfo{ExeName: "python"}, true},
		{"python2", &ProcessInfo{ExeName: "python2"}, true},
		{"python3", &ProcessInfo{ExeName: "python3"}, true},
		{"python3.11", &ProcessInfo{ExeName: "python3.11"}, true},
		{"python3.12", &ProcessInfo{ExeName: "python3.12"}, true},
		{"pypy", &ProcessInfo{ExeName: "pypy"}, true},
		{"pypy3", &ProcessInfo{ExeName: "pypy3"}, true},

		// Python binaries
		{"gunicorn", &ProcessInfo{ExeName: "gunicorn"}, true},
		{"uvicorn", &ProcessInfo{ExeName: "uvicorn"}, true},
		{"celery", &ProcessInfo{ExeName: "celery"}, true},
		{"flask", &ProcessInfo{ExeName: "flask"}, true},
		{"django-admin", &ProcessInfo{ExeName: "django-admin"}, true},

		// Command patterns
		{"python in cmdline", &ProcessInfo{ExeName: "sh", CmdLine: "python app.py"}, true},
		{"python3 in cmdline", &ProcessInfo{ExeName: "sh", CmdLine: "python3 manage.py runserver"}, true},
		{"gunicorn in cmdline", &ProcessInfo{ExeName: "sh", CmdLine: "gunicorn myapp:app"}, true},
		{"manage.py runserver", &ProcessInfo{ExeName: "sh", CmdLine: "manage.py runserver"}, true},
		{"flask run in cmdline", &ProcessInfo{ExeName: "sh", CmdLine: "flask run --port=5000"}, true},

		// .py fallback
		{"py file in cmdline", &ProcessInfo{ExeName: "sh", CmdLine: "/opt/script.py"}, true},

		// Negative cases
		{"java", &ProcessInfo{ExeName: "java", CmdLine: "java -jar app.jar"}, false},
		{"node", &ProcessInfo{ExeName: "node", CmdLine: "node server.js"}, false},
		{"pythonista not matched", &ProcessInfo{ExeName: "sh", CmdLine: "pythonista"}, false},
		{"empty", &ProcessInfo{ExeName: "", CmdLine: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.Detect(tt.proc)
			if got != tt.want {
				t.Errorf("Detect(%+v) = %v, want %v", tt.proc, got, tt.want)
			}
		})
	}
}
