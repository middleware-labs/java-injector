package discovery

import "context"

const dockerSocket = "/var/run/docker.sock"

type dockerClient struct {
	unixSocketClient
}

func newDockerClient() *dockerClient {
	return &dockerClient{
		unixSocketClient: newUnixSocketClient("docker", dockerSocket),
	}
}

func (d *dockerClient) Name() string { return "docker" }

func (d *dockerClient) Available(ctx context.Context) bool {
	return d.ping(ctx)
}
