package discovery

import (
	"context"
	"testing"
)

func makeContainerProcess(containerID, runtime, serviceName string) *Process {
	return &Process{
		PID:         1000,
		ServiceName: serviceName,
		ContainerInfo: &ContainerInfo{
			IsContainer: true,
			ContainerID: containerID,
			Runtime:     runtime,
		},
		Details: make(map[string]any),
	}
}

func TestContainerResolutionToServiceNamePipeline(t *testing.T) {
	id1 := "aaaa" + "bbbbccccddddeeeeffffaaaabbbbccccddddeeeeffffaaaabbbbccccdddd"
	id2 := "1111" + "2222333344445555666611112222333344445555666611112222333344445555"
	ctx := context.Background()

	t.Run("resolves and promotes", func(t *testing.T) {
		clearGlobalNameCache()

		container1 := makeContainerProcess(id1, "docker/containerd", "java-service")
		container2 := makeContainerProcess(id2, "docker/containerd", "node-service")
		container2.PID = 2000
		nonContainer := &Process{
			PID:         3000,
			ServiceName: "host-app",
			Details:     make(map[string]any),
		}

		client := &mockContainerClient{
			name: "docker",
			results: map[string]ContainerMeta{
				id1: {ID: id1, Name: "billing-api"},
				id2: {ID: id2, Name: "frontend"},
			},
		}

		processes := []*Process{container1, container2, nonContainer}
		batchResolveContainerNames(ctx, processes, []ContainerClient{client})
		applyContainerServiceNames(processes)

		if container1.ContainerInfo.ContainerName != "billing-api" {
			t.Errorf("container1 name = %q, want %q", container1.ContainerInfo.ContainerName, "billing-api")
		}
		if container1.ServiceName != "billing-api" {
			t.Errorf("container1 service = %q, want %q", container1.ServiceName, "billing-api")
		}

		if container2.ContainerInfo.ContainerName != "frontend" {
			t.Errorf("container2 name = %q, want %q", container2.ContainerInfo.ContainerName, "frontend")
		}
		if container2.ServiceName != "frontend" {
			t.Errorf("container2 service = %q, want %q", container2.ServiceName, "frontend")
		}

		if nonContainer.ServiceName != "host-app" {
			t.Errorf("non-container service = %q, want %q", nonContainer.ServiceName, "host-app")
		}
	})

	t.Run("pre-named container preserved", func(t *testing.T) {
		clearGlobalNameCache()

		proc := makeContainerProcess(id1, "docker/containerd", "original-service")
		proc.ContainerInfo.ContainerName = "already-named"

		client := &mockContainerClient{
			name: "docker",
			results: map[string]ContainerMeta{
				id1: {ID: id1, Name: "different-name"},
			},
		}

		processes := []*Process{proc}
		batchResolveContainerNames(ctx, processes, []ContainerClient{client})
		applyContainerServiceNames(processes)

		if proc.ContainerInfo.ContainerName != "already-named" {
			t.Errorf("container name = %q, want %q", proc.ContainerInfo.ContainerName, "already-named")
		}
		if proc.ServiceName != "already-named" {
			t.Errorf("service name = %q, want %q", proc.ServiceName, "already-named")
		}
	})

	t.Run("cache hit on second pass", func(t *testing.T) {
		clearGlobalNameCache()

		proc := makeContainerProcess(id1, "docker/containerd", "java-service")

		client := &mockContainerClient{
			name: "docker",
			results: map[string]ContainerMeta{
				id1: {ID: id1, Name: "billing-api"},
			},
		}

		// First pass: resolves via client
		processes := []*Process{proc}
		batchResolveContainerNames(ctx, processes, []ContainerClient{client})
		applyContainerServiceNames(processes)

		if len(client.calls) != 1 {
			t.Fatalf("first pass: expected 1 client call, got %d", len(client.calls))
		}

		// Reset names on process but keep container ID
		proc.ContainerInfo.ContainerName = ""
		proc.ServiceName = "java-service"

		// Second pass: should use cache, not call client
		batchResolveContainerNames(ctx, processes, []ContainerClient{client})
		applyContainerServiceNames(processes)

		if len(client.calls) != 1 {
			t.Errorf("second pass: expected still 1 client call (cache hit), got %d", len(client.calls))
		}
		if proc.ServiceName != "billing-api" {
			t.Errorf("service name after cache hit = %q, want %q", proc.ServiceName, "billing-api")
		}
	})

	t.Run("mixed runtimes", func(t *testing.T) {
		clearGlobalNameCache()

		dockerProc := makeContainerProcess(id1, "docker/containerd", "docker-svc")
		podmanProc := makeContainerProcess(id2, "podman", "podman-svc")
		podmanProc.PID = 2000

		dockerClient := &mockContainerClient{
			name:    "docker",
			results: map[string]ContainerMeta{id1: {ID: id1, Name: "docker-billing"}},
		}
		podmanClient := &mockContainerClient{
			name:    "podman",
			results: map[string]ContainerMeta{id2: {ID: id2, Name: "podman-frontend"}},
		}

		processes := []*Process{dockerProc, podmanProc}
		batchResolveContainerNames(ctx, processes, []ContainerClient{dockerClient, podmanClient})
		applyContainerServiceNames(processes)

		if dockerProc.ServiceName != "docker-billing" {
			t.Errorf("docker proc service = %q, want %q", dockerProc.ServiceName, "docker-billing")
		}
		if podmanProc.ServiceName != "podman-frontend" {
			t.Errorf("podman proc service = %q, want %q", podmanProc.ServiceName, "podman-frontend")
		}

		if len(dockerClient.calls) != 1 {
			t.Errorf("docker client calls = %d, want 1", len(dockerClient.calls))
		}
		if len(podmanClient.calls) != 1 {
			t.Errorf("podman client calls = %d, want 1", len(podmanClient.calls))
		}
	})

	t.Run("unresolvable container unchanged", func(t *testing.T) {
		clearGlobalNameCache()

		proc := makeContainerProcess(id1, "docker/containerd", "original-name")

		client := &mockContainerClient{
			name:    "docker",
			results: map[string]ContainerMeta{}, // empty: ID not found
		}

		processes := []*Process{proc}
		batchResolveContainerNames(ctx, processes, []ContainerClient{client})
		applyContainerServiceNames(processes)

		if proc.ContainerInfo.ContainerName != "" {
			t.Errorf("container name = %q, want empty", proc.ContainerInfo.ContainerName)
		}
		if proc.ServiceName != "original-name" {
			t.Errorf("service name = %q, want %q", proc.ServiceName, "original-name")
		}
	})
}
