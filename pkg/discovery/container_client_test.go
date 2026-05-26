package discovery

import (
	"context"
	"fmt"
	"testing"
)

// mockContainerClient implements ContainerClient for testing.
type mockContainerClient struct {
	name    string
	results map[string]ContainerMeta // ID → meta
	err     error                    // returned by InspectBatch
	calls   [][]string               // records each InspectBatch call
}

func (m *mockContainerClient) Name() string { return m.name }
func (m *mockContainerClient) Available(_ context.Context) bool { return true }
func (m *mockContainerClient) InspectBatch(_ context.Context, ids []string) (map[string]ContainerMeta, error) {
	m.calls = append(m.calls, ids)
	if m.err != nil {
		return nil, m.err
	}
	out := make(map[string]ContainerMeta)
	for _, id := range ids {
		if meta, ok := m.results[id]; ok {
			out[id] = meta
		}
	}
	return out, nil
}

// clearGlobalNameCache resets the global container name cache between tests.
func clearGlobalNameCache() {
	globalNameMu.Lock()
	defer globalNameMu.Unlock()
	globalNameCache = make(map[string]containerNameEntry)
}

func TestBatchResolveContainerNames(t *testing.T) {
	id1 := "aaaa" + "bbbbccccddddeeeeffffaaaabbbbccccddddeeeeffffaaaabbbbccccdddd"
	id2 := "1111" + "2222333344445555666611112222333344445555666611112222333344445555"

	t.Run("resolves names via matching client", func(t *testing.T) {
		clearGlobalNameCache()

		client := &mockContainerClient{
			name: "docker",
			results: map[string]ContainerMeta{
				id1: {ID: id1, Name: "billing-api"},
				id2: {ID: id2, Name: "gateway"},
			},
		}

		procs := []*Process{
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id1, Runtime: "docker/containerd"}},
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id2, Runtime: "docker/containerd"}},
		}

		batchResolveContainerNames(context.Background(), procs, []ContainerClient{client})

		if procs[0].ContainerInfo.ContainerName != "billing-api" {
			t.Errorf("proc[0] name = %q, want %q", procs[0].ContainerInfo.ContainerName, "billing-api")
		}
		if procs[1].ContainerInfo.ContainerName != "gateway" {
			t.Errorf("proc[1] name = %q, want %q", procs[1].ContainerInfo.ContainerName, "gateway")
		}
	})

	t.Run("deduplicates IDs in same runtime", func(t *testing.T) {
		clearGlobalNameCache()

		client := &mockContainerClient{
			name:    "docker",
			results: map[string]ContainerMeta{id1: {ID: id1, Name: "billing-api"}},
		}

		procs := []*Process{
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id1, Runtime: "docker"}},
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id1, Runtime: "docker"}},
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id1, Runtime: "docker"}},
		}

		batchResolveContainerNames(context.Background(), procs, []ContainerClient{client})

		if len(client.calls) != 1 {
			t.Fatalf("expected 1 InspectBatch call, got %d", len(client.calls))
		}
		if len(client.calls[0]) != 1 {
			t.Errorf("expected 1 ID in batch, got %d", len(client.calls[0]))
		}
		for i, p := range procs {
			if p.ContainerInfo.ContainerName != "billing-api" {
				t.Errorf("proc[%d] name = %q, want %q", i, p.ContainerInfo.ContainerName, "billing-api")
			}
		}
	})

	t.Run("skips processes with name already set", func(t *testing.T) {
		clearGlobalNameCache()

		client := &mockContainerClient{
			name:    "docker",
			results: map[string]ContainerMeta{},
		}

		procs := []*Process{
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id1, ContainerName: "already-named", Runtime: "docker"}},
		}

		batchResolveContainerNames(context.Background(), procs, []ContainerClient{client})

		if len(client.calls) != 0 {
			t.Errorf("expected 0 InspectBatch calls for already-named container, got %d", len(client.calls))
		}
		if procs[0].ContainerInfo.ContainerName != "already-named" {
			t.Errorf("name should be unchanged, got %q", procs[0].ContainerInfo.ContainerName)
		}
	})

	t.Run("skips non-container processes", func(t *testing.T) {
		clearGlobalNameCache()

		client := &mockContainerClient{name: "docker", results: map[string]ContainerMeta{}}

		procs := []*Process{
			{ContainerInfo: nil},
			{ContainerInfo: &ContainerInfo{IsContainer: false}},
		}

		batchResolveContainerNames(context.Background(), procs, []ContainerClient{client})

		if len(client.calls) != 0 {
			t.Errorf("expected 0 calls for non-container processes, got %d", len(client.calls))
		}
	})

	t.Run("routes runtimes to correct client", func(t *testing.T) {
		clearGlobalNameCache()

		dockerClient := &mockContainerClient{
			name:    "docker",
			results: map[string]ContainerMeta{id1: {ID: id1, Name: "docker-app"}},
		}
		podmanClient := &mockContainerClient{
			name:    "podman",
			results: map[string]ContainerMeta{id2: {ID: id2, Name: "podman-app"}},
		}

		procs := []*Process{
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id1, Runtime: "docker/containerd"}},
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id2, Runtime: "podman"}},
		}

		batchResolveContainerNames(context.Background(), procs, []ContainerClient{dockerClient, podmanClient})

		if procs[0].ContainerInfo.ContainerName != "docker-app" {
			t.Errorf("docker proc name = %q, want %q", procs[0].ContainerInfo.ContainerName, "docker-app")
		}
		if procs[1].ContainerInfo.ContainerName != "podman-app" {
			t.Errorf("podman proc name = %q, want %q", procs[1].ContainerInfo.ContainerName, "podman-app")
		}
		if len(dockerClient.calls) != 1 {
			t.Errorf("docker client: expected 1 call, got %d", len(dockerClient.calls))
		}
		if len(podmanClient.calls) != 1 {
			t.Errorf("podman client: expected 1 call, got %d", len(podmanClient.calls))
		}
	})

	t.Run("cache hit skips API call", func(t *testing.T) {
		clearGlobalNameCache()
		cacheContainerName(id1, "cached-name")

		client := &mockContainerClient{name: "docker", results: map[string]ContainerMeta{}}

		procs := []*Process{
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id1, Runtime: "docker"}},
		}

		batchResolveContainerNames(context.Background(), procs, []ContainerClient{client})

		if procs[0].ContainerInfo.ContainerName != "cached-name" {
			t.Errorf("name = %q, want %q (from cache)", procs[0].ContainerInfo.ContainerName, "cached-name")
		}
		if len(client.calls) != 0 {
			t.Errorf("expected 0 API calls when cache hit, got %d", len(client.calls))
		}
	})

	t.Run("InspectBatch error is graceful", func(t *testing.T) {
		clearGlobalNameCache()

		client := &mockContainerClient{
			name: "docker",
			err:  fmt.Errorf("connection refused"),
		}

		procs := []*Process{
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id1, Runtime: "docker"}},
		}

		batchResolveContainerNames(context.Background(), procs, []ContainerClient{client})

		if procs[0].ContainerInfo.ContainerName != "" {
			t.Errorf("name should be empty on error, got %q", procs[0].ContainerInfo.ContainerName)
		}
	})

	t.Run("no clients is a no-op", func(t *testing.T) {
		procs := []*Process{
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id1, Runtime: "docker"}},
		}

		batchResolveContainerNames(context.Background(), procs, nil)

		if procs[0].ContainerInfo.ContainerName != "" {
			t.Errorf("name should be empty with no clients, got %q", procs[0].ContainerInfo.ContainerName)
		}
	})

	t.Run("resolved names are cached for future calls", func(t *testing.T) {
		clearGlobalNameCache()

		client := &mockContainerClient{
			name:    "docker",
			results: map[string]ContainerMeta{id1: {ID: id1, Name: "billing-api"}},
		}

		procs := []*Process{
			{ContainerInfo: &ContainerInfo{IsContainer: true, ContainerID: id1, Runtime: "docker"}},
		}

		batchResolveContainerNames(context.Background(), procs, []ContainerClient{client})

		// Verify it was cached
		name, hit := getCachedContainerName(id1)
		if !hit {
			t.Fatal("expected cache hit after resolve")
		}
		if name != "billing-api" {
			t.Errorf("cached name = %q, want %q", name, "billing-api")
		}
	})
}

func TestApplyContainerServiceNames(t *testing.T) {
	t.Run("container name overrides service name", func(t *testing.T) {
		procs := []*Process{
			{
				ServiceName:   "old-name",
				ContainerInfo: &ContainerInfo{IsContainer: true, ContainerName: "billing-api"},
			},
		}

		applyContainerServiceNames(procs)

		if procs[0].ServiceName != "billing-api" {
			t.Errorf("ServiceName = %q, want %q", procs[0].ServiceName, "billing-api")
		}
	})

	t.Run("empty container name does not override", func(t *testing.T) {
		procs := []*Process{
			{
				ServiceName:   "keep-this",
				ContainerInfo: &ContainerInfo{IsContainer: true, ContainerName: ""},
			},
		}

		applyContainerServiceNames(procs)

		if procs[0].ServiceName != "keep-this" {
			t.Errorf("ServiceName = %q, want %q", procs[0].ServiceName, "keep-this")
		}
	})

	t.Run("non-container process untouched", func(t *testing.T) {
		procs := []*Process{
			{ServiceName: "keep-this", ContainerInfo: nil},
			{ServiceName: "also-keep", ContainerInfo: &ContainerInfo{IsContainer: false}},
		}

		applyContainerServiceNames(procs)

		if procs[0].ServiceName != "keep-this" {
			t.Errorf("proc[0] ServiceName = %q, want %q", procs[0].ServiceName, "keep-this")
		}
		if procs[1].ServiceName != "also-keep" {
			t.Errorf("proc[1] ServiceName = %q, want %q", procs[1].ServiceName, "also-keep")
		}
	})

	t.Run("mixed processes only containers updated", func(t *testing.T) {
		procs := []*Process{
			{ServiceName: "original", ContainerInfo: &ContainerInfo{IsContainer: true, ContainerName: "new-name"}},
			{ServiceName: "system-proc", ContainerInfo: nil},
			{ServiceName: "unnamed-container", ContainerInfo: &ContainerInfo{IsContainer: true, ContainerName: ""}},
		}

		applyContainerServiceNames(procs)

		if procs[0].ServiceName != "new-name" {
			t.Errorf("proc[0] = %q, want %q", procs[0].ServiceName, "new-name")
		}
		if procs[1].ServiceName != "system-proc" {
			t.Errorf("proc[1] = %q, want %q", procs[1].ServiceName, "system-proc")
		}
		if procs[2].ServiceName != "unnamed-container" {
			t.Errorf("proc[2] = %q, want %q", procs[2].ServiceName, "unnamed-container")
		}
	})
}
