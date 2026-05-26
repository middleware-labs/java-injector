package discovery

import (
	"strings"
	"testing"
)

const fakeID = "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"

func TestParseCgroupContent(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		wantContainer bool
		wantRuntime   string
		wantID        string
	}{
		// --- Docker & Containerd ---
		{
			name:          "docker v1 cgroup",
			content:       "12:memory:/docker/" + fakeID + "\n",
			wantContainer: true,
			wantRuntime:   "docker/containerd",
			wantID:        fakeID,
		},
		{
			name:          "containerd v1 cgroup",
			content:       "12:memory:/containerd/" + fakeID + "\n",
			wantContainer: true,
			wantRuntime:   "docker/containerd",
			wantID:        fakeID,
		},
		{
			name:          "docker v2 systemd cgroup driver",
			content:       "0::/system.slice/docker-" + fakeID + ".scope\n",
			wantContainer: true,
			wantRuntime:   "docker/containerd",
			wantID:        fakeID,
		},
		{
			name:          "containerd v2 systemd cgroup driver",
			content:       "0::/system.slice/containerd-" + fakeID + ".scope\n",
			wantContainer: true,
			wantRuntime:   "docker/containerd",
			wantID:        fakeID,
		},
		{
			name: "docker v1 multi-controller cgroup",
			content: "12:memory:/docker/" + fakeID + "\n" +
				"11:cpu:/docker/" + fakeID + "\n" +
				"1:name=systemd:/docker/" + fakeID + "\n",
			wantContainer: true,
			wantRuntime:   "docker/containerd",
			wantID:        fakeID,
		},

		// --- Kubernetes ---
		{
			name:          "kubernetes cri-containerd matches docker regex first",
			content:       "0::/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podabc123.slice/cri-containerd-" + fakeID + ".scope\n",
			wantContainer: true,
			wantRuntime:   "docker/containerd",
			wantID:        fakeID,
		},
		{
			name:          "kubernetes crio",
			content:       "0::/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podUID.slice/crio-" + fakeID + ".scope\n",
			wantContainer: true,
			wantRuntime:   "kubernetes",
			wantID:        fakeID,
		},
		{
			name:          "kubernetes kubepods path",
			content:       "0::/kubepods/pod12345678-abcd-efgh-ijkl-123456789012/" + fakeID + "\n",
			wantContainer: true,
			wantRuntime:   "kubernetes",
			wantID:        fakeID,
		},

		// --- Podman ---
		{
			name:          "podman libpod scope",
			content:       "0::/user.slice/user-1000.slice/user@1000.service/libpod-" + fakeID + ".scope\n",
			wantContainer: true,
			wantRuntime:   "podman",
			wantID:        fakeID,
		},
		{
			name:          "podman libpod path",
			content:       "0::/libpod-" + fakeID + "\n",
			wantContainer: true,
			wantRuntime:   "podman",
			wantID:        fakeID,
		},

		// --- LXC ---
		{
			name:          "lxc container",
			content:       "0::/lxc/my-container\n",
			wantContainer: true,
			wantRuntime:   "lxc",
			wantID:        "my-container",
		},
		{
			name:          "lxc with nested path",
			content:       "12:memory:/lxc/webserver\n",
			wantContainer: true,
			wantRuntime:   "lxc",
			wantID:        "webserver",
		},

		// --- Generic scope fallback ---
		{
			name:          "unknown runtime scope",
			content:       "0::/myruntime.scope\n",
			wantContainer: true,
			wantRuntime:   "generic-container",
			wantID:        "",
		},

		// --- NOT a container ---
		{
			name:          "init scope is not container",
			content:       "0::/init.scope\n",
			wantContainer: false,
		},
		{
			name:          "user slice is not container",
			content:       "0::/user.slice/user-1000.slice/session-42.scope\n",
			wantContainer: false,
		},
		{
			name:          "systemd service is not container",
			content:       "0::/system.slice/sshd.service\n",
			wantContainer: false,
		},
		{
			name:          "plain system slice no scope",
			content:       "0::/system.slice/nginx.service\n",
			wantContainer: false,
		},
		{
			name:          "empty content",
			content:       "",
			wantContainer: false,
		},

		// --- Priority: Docker wins over generic scope ---
		{
			name: "docker detected before generic scope fallback",
			content: "0::/system.slice/docker-" + fakeID + ".scope\n" +
				"1:name=systemd:/system.slice/docker-" + fakeID + ".scope\n",
			wantContainer: true,
			wantRuntime:   "docker/containerd",
			wantID:        fakeID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCgroupContent(tt.content)

			if got.IsContainer != tt.wantContainer {
				t.Errorf("IsContainer = %v, want %v", got.IsContainer, tt.wantContainer)
			}
			if tt.wantContainer {
				if got.Runtime != tt.wantRuntime {
					t.Errorf("Runtime = %q, want %q", got.Runtime, tt.wantRuntime)
				}
				if tt.wantID != "" && got.ContainerID != tt.wantID {
					t.Errorf("ContainerID = %q, want %q", got.ContainerID, tt.wantID)
				}
			}
		})
	}
}

func TestParseCgroupContent_IDLength(t *testing.T) {
	shortID := strings.Repeat("ab", 16) // 32 chars — too short for docker regex

	t.Run("short ID does not match docker regex but hits generic scope", func(t *testing.T) {
		content := "0::/system.slice/docker-" + shortID + ".scope\n"
		got := parseCgroupContent(content)
		if !got.IsContainer {
			t.Fatal("expected IsContainer=true from generic scope fallback")
		}
		if got.Runtime != "generic-container" {
			t.Errorf("Runtime = %q, want %q", got.Runtime, "generic-container")
		}
	})

	t.Run("short ID without scope is not container", func(t *testing.T) {
		content := "0::/system.slice/docker-" + shortID + ".service\n"
		got := parseCgroupContent(content)
		if got.IsContainer {
			t.Error("expected IsContainer=false for short ID without .scope")
		}
	})
}

func TestParseCgroupUnitContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantName string
		wantOK   bool
	}{
		// cgroup v2
		{
			name:     "v2 system service",
			content:  "0::/system.slice/flask-app.service\n",
			wantName: "flask-app",
			wantOK:   true,
		},
		{
			name:     "v2 nginx service",
			content:  "0::/system.slice/nginx.service\n",
			wantName: "nginx",
			wantOK:   true,
		},

		// cgroup v1 with name=systemd
		{
			name:     "v1 systemd controller",
			content:  "1:name=systemd:/system.slice/billing-api.service\n",
			wantName: "billing-api",
			wantOK:   true,
		},

		// Nested slices
		{
			name:     "nested system slice",
			content:  "0::/system.slice/system-serial\\x2dgetty.slice/serial-getty@ttyS0.service\n",
			wantName: "serial-getty@ttyS0",
			wantOK:   true,
		},

		// Filtered: user@ services
		{
			name:     "user@ service filtered",
			content:  "0::/user.slice/user@1000.service\n",
			wantName: "",
			wantOK:   false,
		},

		// Filtered: app- transient (desktop units)
		{
			name:     "app- transient desktop unit filtered",
			content:  "0::/app-gnome-terminal-12345.service\n",
			wantName: "",
			wantOK:   false,
		},

		// No .service suffix
		{
			name:     "scope not a service",
			content:  "0::/system.slice/docker.scope\n",
			wantName: "",
			wantOK:   false,
		},

		// Docker container cgroup — no service
		{
			name:     "docker container no service",
			content:  "0::/docker/" + fakeID + "\n",
			wantName: "",
			wantOK:   false,
		},

		// Empty
		{
			name:     "empty content",
			content:  "",
			wantName: "",
			wantOK:   false,
		},

		// Non-systemd controller lines skipped
		{
			name:     "memory controller only skipped",
			content:  "12:memory:/system.slice/nginx.service\n",
			wantName: "",
			wantOK:   false,
		},

		// Multi-line: v1 has both systemd and memory controllers
		{
			name: "multi-line picks systemd controller",
			content: "12:memory:/system.slice/nginx.service\n" +
				"1:name=systemd:/system.slice/nginx.service\n",
			wantName: "nginx",
			wantOK:   true,
		},

		// Innermost .service wins (scanned from end)
		{
			name:     "innermost service segment wins",
			content:  "0::/system.slice/outer.service/inner.service\n",
			wantName: "inner",
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, ok := parseCgroupUnitContent(tt.content)

			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if name != tt.wantName {
				t.Errorf("name = %q, want %q", name, tt.wantName)
			}
		})
	}
}
