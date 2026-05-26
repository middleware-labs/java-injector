package otelinject

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOBIConfigMultiOperationRoundTrip(t *testing.T) {
	baseYAML := `otel_traces_exporter: otlp
target_url: https://example.com
discovery:
  poll_interval: 5s
`
	cfg := parseConfigString(t, baseYAML)

	// Add Java selector
	if _, err := cfg.AddSelector(OBISelector{
		Name: "billing-api", Languages: "java", OpenPorts: "8080",
	}); err != nil {
		t.Fatalf("add java: %v", err)
	}

	// Add Node selector
	if _, err := cfg.AddSelector(OBISelector{
		Name: "frontend", Languages: "nodejs", OpenPorts: "3000,3001",
	}); err != nil {
		t.Fatalf("add node: %v", err)
	}

	// Add Python selector
	if _, err := cfg.AddSelector(OBISelector{
		Name: "ml-pipeline", Languages: "python",
	}); err != nil {
		t.Fatalf("add python: %v", err)
	}

	// Remove Java selector
	if !cfg.RemoveSelector("billing-api") {
		t.Fatal("expected RemoveSelector(billing-api) to return true")
	}

	// Add Go selector
	if _, err := cfg.AddSelector(OBISelector{
		Name: "gateway", Languages: "go", OpenPorts: "443",
	}); err != nil {
		t.Fatalf("add go: %v", err)
	}

	// Overwrite Node selector with new ports
	overwritten, err := cfg.AddSelector(OBISelector{
		Name: "frontend", Languages: "nodejs", OpenPorts: "4000,4001",
	})
	if err != nil {
		t.Fatalf("overwrite node: %v", err)
	}
	if !overwritten {
		t.Fatal("expected overwrite=true when updating frontend")
	}

	// Write to disk and read back
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := cfg.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}

	reloaded, err := ReadOBIConfig(path)
	if err != nil {
		t.Fatalf("ReadOBIConfig: %v", err)
	}

	// Verify selectors
	sels := reloaded.ListSelectors()
	if len(sels) != 3 {
		t.Fatalf("expected 3 selectors, got %d", len(sels))
	}

	want := map[string]OBISelector{
		"ml-pipeline": {Name: "ml-pipeline", Languages: "python"},
		"gateway":     {Name: "gateway", Languages: "go", OpenPorts: "443"},
		"frontend":    {Name: "frontend", Languages: "nodejs", OpenPorts: "4000,4001"},
	}
	for _, sel := range sels {
		expected, ok := want[sel.Name]
		if !ok {
			t.Errorf("unexpected selector: %q", sel.Name)
			continue
		}
		if sel.Languages != expected.Languages {
			t.Errorf("selector %q: languages = %q, want %q", sel.Name, sel.Languages, expected.Languages)
		}
		if sel.OpenPorts != expected.OpenPorts {
			t.Errorf("selector %q: open_ports = %q, want %q", sel.Name, sel.OpenPorts, expected.OpenPorts)
		}
	}

	// Verify billing-api was removed
	if reloaded.HasSelector("billing-api") {
		t.Error("billing-api should have been removed")
	}

	// Verify unrelated sections preserved
	data, _ := os.ReadFile(path)
	output := string(data)
	for _, key := range []string{"otel_traces_exporter", "target_url", "poll_interval"} {
		if !strings.Contains(output, key) {
			t.Errorf("output missing preserved key %q", key)
		}
	}
}

func TestOBIConfigSequentialBulkOperations(t *testing.T) {
	cfg := NewEmptyOBIConfig()

	const count = 50
	for i := range count {
		sel := OBISelector{
			Name:      fmt.Sprintf("svc-%d", i),
			Languages: "java",
			OpenPorts: fmt.Sprintf("%d", 8000+i),
		}
		if _, err := cfg.AddSelector(sel); err != nil {
			t.Fatalf("AddSelector(%d): %v", i, err)
		}
	}

	sels := cfg.ListSelectors()
	if len(sels) != count {
		t.Fatalf("expected %d selectors, got %d", count, len(sels))
	}

	// Remove every other selector
	for i := 0; i < count; i += 2 {
		name := fmt.Sprintf("svc-%d", i)
		if !cfg.RemoveSelector(name) {
			t.Errorf("RemoveSelector(%q) returned false", name)
		}
	}

	sels = cfg.ListSelectors()
	if len(sels) != count/2 {
		t.Fatalf("after removal: expected %d selectors, got %d", count/2, len(sels))
	}

	// Verify remaining are odd-numbered
	for _, sel := range sels {
		var idx int
		fmt.Sscanf(sel.Name, "svc-%d", &idx)
		if idx%2 == 0 {
			t.Errorf("even selector %q should have been removed", sel.Name)
		}
	}
}

func TestOBIConfigNullInstrumentSequence(t *testing.T) {
	yamlContent := `discovery:
  instrument:
`
	cfg := parseConfigString(t, yamlContent)

	if _, err := cfg.AddSelector(OBISelector{
		Name: "my-app", Languages: "java", OpenPorts: "8080",
	}); err != nil {
		t.Fatalf("AddSelector on null instrument: %v", err)
	}

	sels := cfg.ListSelectors()
	if len(sels) != 1 {
		t.Fatalf("expected 1 selector, got %d", len(sels))
	}
	if sels[0].Name != "my-app" {
		t.Errorf("name = %q, want %q", sels[0].Name, "my-app")
	}

	// Round-trip through file
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := cfg.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}

	reloaded, err := ReadOBIConfig(path)
	if err != nil {
		t.Fatalf("ReadOBIConfig: %v", err)
	}

	sels = reloaded.ListSelectors()
	if len(sels) != 1 || sels[0].Name != "my-app" {
		t.Fatalf("after round-trip: unexpected selectors: %+v", sels)
	}
}
