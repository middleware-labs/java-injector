package discovery

import "testing"

func TestSplitImageTag(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantRepo string
		wantTag  string
	}{
		{"simple with tag", "nginx:1.25", "nginx", "1.25"},
		{"registry with tag", "registry.example.com/app:latest", "registry.example.com/app", "latest"},
		{"no tag", "nginx", "nginx", ""},
		{"registry port with tag", "registry.example.com:5000/app:v1", "registry.example.com:5000/app", "v1"},
		{"registry port no tag", "registry.example.com:5000/app", "registry.example.com:5000/app", ""},
		{"latest tag", "myapp:latest", "myapp", "latest"},
		{"empty", "", "", ""},
		{"sha digest splits at colon", "nginx@sha256:abc123", "nginx@sha256", "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, tag := splitImageTag(tt.input)
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
			if tag != tt.wantTag {
				t.Errorf("tag = %q, want %q", tag, tt.wantTag)
			}
		})
	}
}

func TestExtractComposeInfo(t *testing.T) {
	tests := []struct {
		name       string
		labels     map[string]string
		wantNil    bool
		wantProj   string
		wantSvc    string
		wantWorkDir string
	}{
		{
			name: "full compose labels",
			labels: map[string]string{
				"com.docker.compose.project":             "myproject",
				"com.docker.compose.service":             "api",
				"com.docker.compose.project.working_dir": "/home/user/myproject",
			},
			wantProj:    "myproject",
			wantSvc:     "api",
			wantWorkDir: "/home/user/myproject",
		},
		{
			name: "project only",
			labels: map[string]string{
				"com.docker.compose.project": "myproject",
			},
			wantProj: "myproject",
			wantSvc:  "",
		},
		{
			name: "service only",
			labels: map[string]string{
				"com.docker.compose.service": "api",
			},
			wantSvc:  "api",
			wantProj: "",
		},
		{
			name:    "no compose labels",
			labels:  map[string]string{"other": "label"},
			wantNil: true,
		},
		{
			name:    "empty labels",
			labels:  map[string]string{},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractComposeInfo(tt.labels)

			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("expected non-nil ComposeInfo")
			}
			if got.Project != tt.wantProj {
				t.Errorf("Project = %q, want %q", got.Project, tt.wantProj)
			}
			if got.Service != tt.wantSvc {
				t.Errorf("Service = %q, want %q", got.Service, tt.wantSvc)
			}
			if tt.wantWorkDir != "" && got.WorkDir != tt.wantWorkDir {
				t.Errorf("WorkDir = %q, want %q", got.WorkDir, tt.wantWorkDir)
			}
		})
	}
}

func TestRuntimeMatchesClient(t *testing.T) {
	tests := []struct {
		name       string
		runtime    string
		clientName string
		want       bool
	}{
		{"docker matches docker", "docker", "docker", true},
		{"docker/containerd matches docker", "docker/containerd", "docker", true},
		{"podman matches podman", "podman", "podman", true},
		{"podman does not match docker", "podman", "docker", false},
		{"docker does not match podman", "docker", "podman", false},
		{"kubernetes does not match docker", "kubernetes", "docker", false},
		{"unknown runtime", "lxc", "docker", false},
		{"unknown client", "docker", "containerd", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runtimeMatchesClient(tt.runtime, tt.clientName)
			if got != tt.want {
				t.Errorf("runtimeMatchesClient(%q, %q) = %v, want %v", tt.runtime, tt.clientName, got, tt.want)
			}
		})
	}
}
