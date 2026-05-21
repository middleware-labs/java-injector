package otelinject

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/middleware-labs/java-injector/pkg/discovery"
)

// groupSettingsForTest replays the grouping logic from DiscoverServices
// on a pre-built slice of ServiceSettings. It skips the systemd drop-in
// os.Stat check (hardcoded path, not testable) but includes the OBI
// config instrumentation check.
func groupSettingsForTest(settings []discovery.ServiceSetting, obiCfg *OBIConfig) []ServiceEntry {
	type group struct {
		representative discovery.ServiceSetting
		instances      []InstanceInfo
		ports          map[int]struct{}
	}
	groups := make(map[string]*group)
	var order []string

	for _, setting := range settings {
		fp := setting.Fingerprint
		if fp == "" {
			fp = setting.Key
		}

		g, exists := groups[fp]
		if !exists {
			g = &group{
				representative: setting,
				ports:          make(map[int]struct{}),
			}
			groups[fp] = g
			order = append(order, fp)
		}

		if len(setting.Instances) > 0 {
			for _, ri := range setting.Instances {
				inst := InstanceInfo{
					PID:    ri.PID,
					Owner:  ri.Owner,
					Status: ri.Status,
				}
				for _, l := range ri.Listeners {
					inst.Ports = append(inst.Ports, int(l.Port))
					g.ports[int(l.Port)] = struct{}{}
				}
				sort.Ints(inst.Ports)
				g.instances = append(g.instances, inst)
			}
		} else {
			inst := InstanceInfo{
				PID:    setting.PID,
				Owner:  setting.Owner,
				Status: setting.Status,
			}
			for _, l := range setting.Listeners {
				inst.Ports = append(inst.Ports, int(l.Port))
				g.ports[int(l.Port)] = struct{}{}
			}
			sort.Ints(inst.Ports)
			g.instances = append(g.instances, inst)
		}
	}

	entries := make([]ServiceEntry, 0, len(groups))
	for _, fp := range order {
		g := groups[fp]
		s := g.representative

		ports := make([]int, 0, len(g.ports))
		for p := range g.ports {
			ports = append(ports, p)
		}
		sort.Ints(ports)

		entry := ServiceEntry{
			Fingerprint:     fp,
			ServiceName:     s.ServiceName,
			Language:        s.Language,
			ServiceType:     s.ServiceType,
			SystemdUnit:     s.SystemdUnit,
			IntegrationType: s.IntegrationType,
			Ports:           ports,
			Instances:       g.instances,
		}

		if obiCfg != nil && obiCfg.HasSelector(s.ServiceName) {
			entry.Instrumented = true
			entry.InstrumentedVia = "obi"
		}

		if !entry.Instrumented && s.Instrumented {
			entry.Instrumented = true
			entry.InstrumentedVia = s.AgentType
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Language != entries[j].Language {
			return entries[i].Language < entries[j].Language
		}
		return entries[i].ServiceName < entries[j].ServiceName
	})

	return entries
}

func TestServiceEntryGroupingFromSettings(t *testing.T) {
	t.Run("replicas merged by fingerprint", func(t *testing.T) {
		settings := []discovery.ServiceSetting{
			{ServiceName: "billing-api", Language: "java", Fingerprint: "fp-abc", PID: 100, Owner: "root", Status: "running",
				Listeners: []discovery.Listener{{Port: 8080, Protocol: "tcp"}}},
			{ServiceName: "billing-api", Language: "java", Fingerprint: "fp-abc", PID: 101, Owner: "root", Status: "running",
				Listeners: []discovery.Listener{{Port: 8081, Protocol: "tcp"}}},
			{ServiceName: "billing-api", Language: "java", Fingerprint: "fp-abc", PID: 102, Owner: "root", Status: "running",
				Listeners: []discovery.Listener{{Port: 8082, Protocol: "tcp"}}},
		}

		entries := groupSettingsForTest(settings, nil)
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}

		e := entries[0]
		if len(e.Instances) != 3 {
			t.Errorf("instances = %d, want 3", len(e.Instances))
		}
		if len(e.Ports) != 3 {
			t.Errorf("merged ports = %d, want 3", len(e.Ports))
		}
		wantPorts := []int{8080, 8081, 8082}
		for i, p := range e.Ports {
			if p != wantPorts[i] {
				t.Errorf("port[%d] = %d, want %d", i, p, wantPorts[i])
			}
		}
	})

	t.Run("standalone service single instance", func(t *testing.T) {
		settings := []discovery.ServiceSetting{
			{ServiceName: "my-cli", Language: "go", Fingerprint: "fp-solo", PID: 500, Owner: "deploy", Status: "running"},
		}

		entries := groupSettingsForTest(settings, nil)
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if len(entries[0].Instances) != 1 {
			t.Errorf("instances = %d, want 1", len(entries[0].Instances))
		}
		if entries[0].Instances[0].PID != 500 {
			t.Errorf("instance PID = %d, want 500", entries[0].Instances[0].PID)
		}
	})

	t.Run("same name different fingerprints separate", func(t *testing.T) {
		settings := []discovery.ServiceSetting{
			{ServiceName: "my-app", Language: "node", Fingerprint: "fp-1", PID: 100},
			{ServiceName: "my-app", Language: "node", Fingerprint: "fp-2", PID: 200},
		}

		entries := groupSettingsForTest(settings, nil)
		if len(entries) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(entries))
		}
	})

	t.Run("obi instrumentation detected", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		os.WriteFile(cfgPath, []byte(`discovery:
  instrument:
    - name: billing-api
      languages: java
      open_ports: "8080"
`), 0644)

		obiCfg, err := ReadOBIConfig(cfgPath)
		if err != nil {
			t.Fatalf("ReadOBIConfig: %v", err)
		}

		settings := []discovery.ServiceSetting{
			{ServiceName: "billing-api", Language: "java", Fingerprint: "fp-1", PID: 100},
			{ServiceName: "other-app", Language: "node", Fingerprint: "fp-2", PID: 200},
		}

		entries := groupSettingsForTest(settings, obiCfg)

		var billingEntry, otherEntry *ServiceEntry
		for i := range entries {
			switch entries[i].ServiceName {
			case "billing-api":
				billingEntry = &entries[i]
			case "other-app":
				otherEntry = &entries[i]
			}
		}

		if billingEntry == nil {
			t.Fatal("billing-api entry not found")
		}
		if !billingEntry.Instrumented {
			t.Error("billing-api should be instrumented")
		}
		if billingEntry.InstrumentedVia != "obi" {
			t.Errorf("billing-api via = %q, want %q", billingEntry.InstrumentedVia, "obi")
		}

		if otherEntry == nil {
			t.Fatal("other-app entry not found")
		}
		if otherEntry.Instrumented {
			t.Error("other-app should NOT be instrumented")
		}
	})

	t.Run("per-instance ports from ReportInstanceInfo", func(t *testing.T) {
		settings := []discovery.ServiceSetting{
			{
				ServiceName: "cluster-app", Language: "node", Fingerprint: "fp-cluster", PID: 100,
				Instances: []discovery.ReportInstanceInfo{
					{PID: 100, Owner: "root", Status: "running",
						Listeners: []discovery.Listener{{Port: 3000, Protocol: "tcp"}}},
					{PID: 200, Owner: "root", Status: "running",
						Listeners: []discovery.Listener{{Port: 3001, Protocol: "tcp"}}},
				},
			},
		}

		entries := groupSettingsForTest(settings, nil)
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}

		e := entries[0]
		if len(e.Instances) != 2 {
			t.Fatalf("instances = %d, want 2", len(e.Instances))
		}
		if len(e.Instances[0].Ports) != 1 || e.Instances[0].Ports[0] != 3000 {
			t.Errorf("instance[0] ports = %v, want [3000]", e.Instances[0].Ports)
		}
		if len(e.Instances[1].Ports) != 1 || e.Instances[1].Ports[0] != 3001 {
			t.Errorf("instance[1] ports = %v, want [3001]", e.Instances[1].Ports)
		}
		if len(e.Ports) != 2 {
			t.Errorf("merged ports count = %d, want 2", len(e.Ports))
		}
	})

	t.Run("entries sorted by language then name", func(t *testing.T) {
		settings := []discovery.ServiceSetting{
			{ServiceName: "z-app", Language: "python", Fingerprint: "fp-3", PID: 300},
			{ServiceName: "a-app", Language: "node", Fingerprint: "fp-2", PID: 200},
			{ServiceName: "billing", Language: "java", Fingerprint: "fp-1", PID: 100},
			{ServiceName: "auth", Language: "java", Fingerprint: "fp-4", PID: 400},
		}

		entries := groupSettingsForTest(settings, nil)
		if len(entries) != 4 {
			t.Fatalf("expected 4 entries, got %d", len(entries))
		}

		// Expected order: java/auth, java/billing, node/a-app, python/z-app
		expected := []struct{ lang, name string }{
			{"java", "auth"},
			{"java", "billing"},
			{"node", "a-app"},
			{"python", "z-app"},
		}
		for i, e := range entries {
			if e.Language != expected[i].lang || e.ServiceName != expected[i].name {
				t.Errorf("entry[%d] = {%s, %s}, want {%s, %s}",
					i, e.Language, e.ServiceName, expected[i].lang, expected[i].name)
			}
		}
	})

	t.Run("empty fingerprint falls back to Key field", func(t *testing.T) {
		settings := []discovery.ServiceSetting{
			{ServiceName: "legacy-app", Language: "java", Key: "key-123", PID: 100,
				Listeners: []discovery.Listener{{Port: 8080, Protocol: "tcp"}}},
			{ServiceName: "legacy-app", Language: "java", Key: "key-123", PID: 101,
				Listeners: []discovery.Listener{{Port: 8081, Protocol: "tcp"}}},
		}

		entries := groupSettingsForTest(settings, nil)
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry (grouped by Key), got %d", len(entries))
		}
		if len(entries[0].Instances) != 2 {
			t.Errorf("instances = %d, want 2", len(entries[0].Instances))
		}
		if entries[0].Fingerprint != "key-123" {
			t.Errorf("fingerprint = %q, want %q", entries[0].Fingerprint, "key-123")
		}
	})
}
