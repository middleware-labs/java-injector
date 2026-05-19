package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// rewriteTransport redirects requests from fake hostnames (e.g. "docker",
// "podman") to the httptest server, preserving the path and query.
type rewriteTransport struct {
	target *url.URL
	inner  http.RoundTripper
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = rt.target.Scheme
	req.URL.Host = rt.target.Host
	return rt.inner.RoundTrip(req)
}

func newTestDockerClient(ts *httptest.Server) *dockerClient {
	u, _ := url.Parse(ts.URL)
	return &dockerClient{httpClient: &http.Client{
		Transport: &rewriteTransport{target: u, inner: http.DefaultTransport},
	}}
}

func newTestPodmanClient(ts *httptest.Server) *podmanClient {
	u, _ := url.Parse(ts.URL)
	return &podmanClient{httpClient: &http.Client{
		Transport: &rewriteTransport{target: u, inner: http.DefaultTransport},
	}}
}

func TestDockerClientInspect(t *testing.T) {
	t.Run("parses standard inspect response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/containers/abc123/json" {
				t.Errorf("unexpected path: %s", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(dockerInspectResponse{
				ID:   "abc123full",
				Name: "/billing-api",
				Config: struct {
					Image  string            `json:"Image"`
					Labels map[string]string `json:"Labels"`
				}{
					Image: "myorg/billing:v1.2.3",
					Labels: map[string]string{
						"com.docker.compose.project": "mystack",
						"com.docker.compose.service": "billing",
					},
				},
			})
		}))
		defer ts.Close()

		client := newTestDockerClient(ts)
		meta, err := client.inspect(context.Background(), "abc123")
		if err != nil {
			t.Fatalf("inspect error: %v", err)
		}

		if meta.Name != "billing-api" {
			t.Errorf("Name = %q, want %q (leading / stripped)", meta.Name, "billing-api")
		}
		if meta.Image != "myorg/billing" {
			t.Errorf("Image = %q, want %q", meta.Image, "myorg/billing")
		}
		if meta.ImageTag != "v1.2.3" {
			t.Errorf("ImageTag = %q, want %q", meta.ImageTag, "v1.2.3")
		}
		if meta.ComposeInfo == nil {
			t.Fatal("ComposeInfo should not be nil")
		}
		if meta.ComposeInfo.Project != "mystack" {
			t.Errorf("ComposeInfo.Project = %q, want %q", meta.ComposeInfo.Project, "mystack")
		}
		if meta.ComposeInfo.Service != "billing" {
			t.Errorf("ComposeInfo.Service = %q, want %q", meta.ComposeInfo.Service, "billing")
		}
	})

	t.Run("name without leading slash", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode(dockerInspectResponse{
				ID:   "def456",
				Name: "no-slash",
			})
		}))
		defer ts.Close()

		client := newTestDockerClient(ts)
		meta, err := client.inspect(context.Background(), "def456")
		if err != nil {
			t.Fatalf("inspect error: %v", err)
		}
		if meta.Name != "no-slash" {
			t.Errorf("Name = %q, want %q", meta.Name, "no-slash")
		}
	})

	t.Run("image without tag", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			resp := dockerInspectResponse{ID: "img1", Name: "/app"}
			resp.Config.Image = "nginx"
			json.NewEncoder(w).Encode(resp)
		}))
		defer ts.Close()

		client := newTestDockerClient(ts)
		meta, err := client.inspect(context.Background(), "img1")
		if err != nil {
			t.Fatalf("inspect error: %v", err)
		}
		if meta.Image != "nginx" {
			t.Errorf("Image = %q, want %q", meta.Image, "nginx")
		}
		if meta.ImageTag != "" {
			t.Errorf("ImageTag = %q, want empty", meta.ImageTag)
		}
	})

	t.Run("no compose labels", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			resp := dockerInspectResponse{ID: "noc1", Name: "/plain"}
			resp.Config.Labels = map[string]string{"other": "label"}
			json.NewEncoder(w).Encode(resp)
		}))
		defer ts.Close()

		client := newTestDockerClient(ts)
		meta, err := client.inspect(context.Background(), "noc1")
		if err != nil {
			t.Fatalf("inspect error: %v", err)
		}
		if meta.ComposeInfo != nil {
			t.Errorf("ComposeInfo should be nil without compose labels, got %+v", meta.ComposeInfo)
		}
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		client := newTestDockerClient(ts)
		_, err := client.inspect(context.Background(), "missing")
		if err == nil {
			t.Error("expected error on 404 response")
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte("not json"))
		}))
		defer ts.Close()

		client := newTestDockerClient(ts)
		_, err := client.inspect(context.Background(), "bad")
		if err == nil {
			t.Error("expected error on malformed JSON")
		}
	})
}

func TestDockerClientInspectBatch(t *testing.T) {
	t.Run("returns available containers, skips missing", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/containers/aaa/json":
				resp := dockerInspectResponse{ID: "aaa", Name: "/app-a"}
				resp.Config.Image = "img-a:latest"
				json.NewEncoder(w).Encode(resp)
			case "/containers/bbb/json":
				w.WriteHeader(http.StatusNotFound)
			case "/containers/ccc/json":
				resp := dockerInspectResponse{ID: "ccc", Name: "/app-c"}
				resp.Config.Image = "img-c:v2"
				json.NewEncoder(w).Encode(resp)
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		client := newTestDockerClient(ts)
		result, err := client.InspectBatch(context.Background(), []string{"aaa", "bbb", "ccc"})
		if err != nil {
			t.Fatalf("InspectBatch error: %v", err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 results (skipping missing), got %d", len(result))
		}
		if result["aaa"].Name != "app-a" {
			t.Errorf("aaa Name = %q, want %q", result["aaa"].Name, "app-a")
		}
		if result["ccc"].Name != "app-c" {
			t.Errorf("ccc Name = %q, want %q", result["ccc"].Name, "app-c")
		}
	})
}

func TestPodmanClientInspect(t *testing.T) {
	t.Run("parses standard inspect response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode(podmanInspectResponse{
				ID:   "pod123",
				Name: "/gateway",
				Config: struct {
					Image  string            `json:"Image"`
					Labels map[string]string `json:"Labels"`
				}{
					Image:  "quay.io/gateway:3.0",
					Labels: map[string]string{},
				},
			})
		}))
		defer ts.Close()

		client := newTestPodmanClient(ts)
		meta, err := client.inspect(context.Background(), "pod123")
		if err != nil {
			t.Fatalf("inspect error: %v", err)
		}
		if meta.Name != "gateway" {
			t.Errorf("Name = %q, want %q", meta.Name, "gateway")
		}
		if meta.Image != "quay.io/gateway" {
			t.Errorf("Image = %q, want %q", meta.Image, "quay.io/gateway")
		}
		if meta.ImageTag != "3.0" {
			t.Errorf("ImageTag = %q, want %q", meta.ImageTag, "3.0")
		}
	})

	t.Run("non-200 status returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		client := newTestPodmanClient(ts)
		_, err := client.inspect(context.Background(), "fail")
		if err == nil {
			t.Error("expected error on 500 response")
		}
	})
}

func TestPodmanClientInspectBatch(t *testing.T) {
	t.Run("returns available containers, skips errors", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/containers/p1/json" {
				resp := podmanInspectResponse{ID: "p1", Name: "/worker"}
				resp.Config.Image = "worker:latest"
				json.NewEncoder(w).Encode(resp)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer ts.Close()

		client := newTestPodmanClient(ts)
		result, err := client.InspectBatch(context.Background(), []string{"p1", "p2"})
		if err != nil {
			t.Fatalf("InspectBatch error: %v", err)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result))
		}
		if result["p1"].Name != "worker" {
			t.Errorf("p1 Name = %q, want %q", result["p1"].Name, "worker")
		}
	})
}
