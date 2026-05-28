//go:build integration

package otelinject

import (
	"context"
	"testing"

	"github.com/middleware-labs/mw-injector/pkg/discovery"
)

func TestE2E_FindAllProcesses(t *testing.T) {
	procs, err := discovery.FindAllProcesses(context.Background())
	if err != nil {
		t.Fatalf("FindAllProcesses failed: %v", err)
	}

	for _, lang := range []discovery.Language{discovery.LangJava, discovery.LangNode, discovery.LangPython} {
		found := procs[lang]
		if len(found) == 0 {
			t.Errorf("expected at least 1 process for %s, found 0", lang)
		}
	}
}

func TestE2E_ListSystemdServices(t *testing.T) {
	services, err := ListSystemdServices()
	if err != nil {
		t.Logf("ListSystemdServices returned partial error (expected): %v", err)
	}

	want := map[string]string{
		"test-java":   "java",
		"test-node":   "node",
		"test-python": "python",
	}

	found := make(map[string]bool)
	for _, svc := range services {
		if expectedLang, ok := want[svc.SystemdUnit]; ok {
			found[svc.SystemdUnit] = true
			if svc.Language != expectedLang {
				t.Errorf("service %s: expected language %q, got %q", svc.SystemdUnit, expectedLang, svc.Language)
			}
			if svc.PID == 0 {
				t.Errorf("service %s: expected non-zero PID", svc.SystemdUnit)
			}
		}
	}

	for unit := range want {
		if !found[unit] {
			t.Errorf("expected to find systemd service %s in discovery results", unit)
		}
	}
}

func TestE2E_DiscoverServices_AllLanguages(t *testing.T) {
	entries, err := DiscoverServices(DiscoverServicesOpts{})
	if err != nil {
		t.Fatalf("DiscoverServices failed: %v", err)
	}

	if len(entries) < 3 {
		t.Errorf("expected at least 3 service entries, got %d", len(entries))
		for _, e := range entries {
			t.Logf("  entry: %s (lang=%s, unit=%s)", e.ServiceName, e.Language, e.SystemdUnit)
		}
	}
}

func TestE2E_Discovery_ServiceNames(t *testing.T) {
	// Service names depend on each handler's priority chain AND container
	// context. Inside this Docker container, IsInContainer() returns true,
	// which affects handlers with OverrideServiceNameOnContainer (Node, Go,
	// Ruby). Java's name comes from the systemd unit; Python falls through
	// to "python-service" because it doesn't check systemd unit name.
	//
	// We verify that each test service is discovered with a non-empty name
	// and the correct SystemdUnit, rather than hardcoding exact names that
	// depend on container context.
	services, err := ListSystemdServices()
	if err != nil {
		t.Logf("ListSystemdServices returned partial error: %v", err)
	}

	wantUnits := map[string]bool{
		"test-java":   false,
		"test-node":   false,
		"test-python": false,
	}

	for _, svc := range services {
		if _, ok := wantUnits[svc.SystemdUnit]; ok {
			wantUnits[svc.SystemdUnit] = true
			t.Logf("unit=%q name=%q language=%q", svc.SystemdUnit, svc.Name, svc.Language)
		}
	}

	for unit, found := range wantUnits {
		if !found {
			t.Errorf("expected systemd unit %q in discovery results", unit)
		}
	}

	// Java should always resolve to "test-java" from the systemd unit name
	// (Java handler checks systemd unit at priority #3, and doesn't override
	// on container).
	for _, svc := range services {
		if svc.SystemdUnit == "test-java" && svc.Name != "test-java" {
			t.Errorf("Java service name: expected %q, got %q", "test-java", svc.Name)
		}
	}
}

func TestE2E_Discovery_Fingerprints(t *testing.T) {
	// Filter to test services by SystemdUnit to avoid the Go test binary
	// (which has an unstable fingerprint since its PID changes between calls).
	testUnits := map[string]bool{
		"test-java": true, "test-node": true, "test-python": true,
	}

	entries1, err := DiscoverServices(DiscoverServicesOpts{})
	if err != nil {
		t.Fatalf("first DiscoverServices failed: %v", err)
	}

	entries2, err := DiscoverServices(DiscoverServicesOpts{})
	if err != nil {
		t.Fatalf("second DiscoverServices failed: %v", err)
	}

	fp1 := make(map[string]string)
	for _, e := range entries1 {
		if !testUnits[e.SystemdUnit] {
			continue
		}
		if e.Fingerprint == "" {
			t.Errorf("service unit=%s name=%s has empty fingerprint", e.SystemdUnit, e.ServiceName)
			continue
		}
		fp1[e.SystemdUnit] = e.Fingerprint
	}

	for _, e := range entries2 {
		if !testUnits[e.SystemdUnit] {
			continue
		}
		if prev, ok := fp1[e.SystemdUnit]; ok {
			if e.Fingerprint != prev {
				t.Errorf("fingerprint for unit %s changed between calls: %s vs %s", e.SystemdUnit, prev, e.Fingerprint)
			}
		}
	}
}

func TestE2E_Discovery_SystemdUnit(t *testing.T) {
	entries, err := DiscoverServices(DiscoverServicesOpts{})
	if err != nil {
		t.Fatalf("DiscoverServices failed: %v", err)
	}

	wantUnits := map[string]bool{
		"test-java":   false,
		"test-node":   false,
		"test-python": false,
	}

	for _, e := range entries {
		if _, ok := wantUnits[e.SystemdUnit]; ok {
			wantUnits[e.SystemdUnit] = true
		}
	}

	for unit, found := range wantUnits {
		if !found {
			t.Errorf("expected SystemdUnit %q in discovery results", unit)
		}
	}
}
