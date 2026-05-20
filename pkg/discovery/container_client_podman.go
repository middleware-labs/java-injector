package discovery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

type podmanClient struct {
	socketPath string
	unixSocketClient
}

func newPodmanClient() *podmanClient {
	sock := findPodmanSocket()
	return &podmanClient{
		socketPath:       sock,
		unixSocketClient: newUnixSocketClient("podman", sock),
	}
}

func findPodmanSocket() string {
	if sock := os.Getenv("CONTAINER_HOST"); sock != "" {
		return strings.TrimPrefix(sock, "unix://")
	}
	if _, err := os.Stat("/run/podman/podman.sock"); err == nil {
		return "/run/podman/podman.sock"
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		sock := filepath.Join(xdg, "podman", "podman.sock")
		if _, err := os.Stat(sock); err == nil {
			return sock
		}
	}
	return "/run/podman/podman.sock"
}

func (p *podmanClient) Name() string { return "podman" }

func (p *podmanClient) Available(ctx context.Context) bool {
	if _, err := os.Stat(p.socketPath); err != nil {
		return false
	}
	return p.ping(ctx)
}
