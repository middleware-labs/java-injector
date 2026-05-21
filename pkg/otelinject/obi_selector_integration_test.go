package otelinject

import (
	"path/filepath"
	"testing"

	"github.com/middleware-labs/java-injector/pkg/discovery"
)

func makeServiceSettingWith(name, serviceType string, ports []uint16) discovery.ServiceSetting {
	s := discovery.ServiceSetting{
		ServiceName: name,
		ServiceType: serviceType,
	}
	for _, p := range ports {
		s.Listeners = append(s.Listeners, discovery.Listener{Port: p, Protocol: "tcp"})
	}
	return s
}

func TestBuildSelectorAndPersistToConfig(t *testing.T) {
	tests := []struct {
		name    string
		service discovery.ServiceSetting
		lang    discovery.Language
		want    OBISelector
	}{
		{
			name: "java with JAR and ports",
			service: func() discovery.ServiceSetting {
				s := makeServiceSettingWith("billing-api", "system", []uint16{8080})
				s.JarFile = "billing-api-1.2.3.jar"
				return s
			}(),
			lang: discovery.LangJava,
			want: OBISelector{
				Name: "billing-api", Languages: "java",
				OpenPorts: "8080", CmdArgs: "*billing-api-1.2.3.jar*",
			},
		},
		{
			name: "java with main class no JAR",
			service: func() discovery.ServiceSetting {
				s := makeServiceSettingWith("billing", "system", nil)
				s.MainClass = "com.example.BillingService"
				return s
			}(),
			lang: discovery.LangJava,
			want: OBISelector{
				Name: "billing", Languages: "java",
				CmdArgs: "*com.example.BillingService*",
			},
		},
		{
			name:    "node with multiple ports",
			service: makeServiceSettingWith("my-api", "system", []uint16{3000, 3001}),
			lang:    discovery.LangNode,
			want: OBISelector{
				Name: "my-api", Languages: "nodejs", OpenPorts: "3000,3001",
			},
		},
		{
			name:    "python standalone",
			service: makeServiceSettingWith("flask-app", "standalone", []uint16{5000}),
			lang:    discovery.LangPython,
			want: OBISelector{
				Name: "flask-app", Languages: "python", OpenPorts: "5000",
			},
		},
		{
			name:    "go with no ports",
			service: makeServiceSettingWith("gateway", "system", nil),
			lang:    discovery.LangGo,
			want:    OBISelector{Name: "gateway", Languages: "go"},
		},
		{
			name:    "rust via docker sets containers_only",
			service: makeServiceSettingWith("my-rust-svc", "docker", []uint16{8443}),
			lang:    discovery.LangRust,
			want: OBISelector{
				Name: "my-rust-svc", Languages: "rust",
				OpenPorts: "8443", ContainersOnly: true,
			},
		},
		{
			name:    "ruby via podman sets containers_only",
			service: makeServiceSettingWith("rails-app", "podman", []uint16{3000}),
			lang:    discovery.LangRuby,
			want: OBISelector{
				Name: "rails-app", Languages: "ruby",
				OpenPorts: "3000", ContainersOnly: true,
			},
		},
	}

	cfg := NewEmptyOBIConfig()

	// Build and add all selectors to the same config
	for _, tt := range tests {
		sel := buildOBISelector(tt.service, tt.lang)
		if _, err := cfg.AddSelector(sel); err != nil {
			t.Fatalf("AddSelector(%s): %v", tt.name, err)
		}
	}

	// Write and read back
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := cfg.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}

	reloaded, err := ReadOBIConfig(path)
	if err != nil {
		t.Fatalf("ReadOBIConfig: %v", err)
	}

	sels := reloaded.ListSelectors()
	if len(sels) != len(tests) {
		t.Fatalf("expected %d selectors, got %d", len(tests), len(sels))
	}

	// Build lookup for verification
	selByName := make(map[string]OBISelector, len(sels))
	for _, s := range sels {
		selByName[s.Name] = s
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := selByName[tt.want.Name]
			if !ok {
				t.Fatalf("selector %q not found after round-trip", tt.want.Name)
			}
			if got.Languages != tt.want.Languages {
				t.Errorf("languages = %q, want %q", got.Languages, tt.want.Languages)
			}
			if got.OpenPorts != tt.want.OpenPorts {
				t.Errorf("open_ports = %q, want %q", got.OpenPorts, tt.want.OpenPorts)
			}
			if got.CmdArgs != tt.want.CmdArgs {
				t.Errorf("cmd_args = %q, want %q", got.CmdArgs, tt.want.CmdArgs)
			}
			if got.ContainersOnly != tt.want.ContainersOnly {
				t.Errorf("containers_only = %v, want %v", got.ContainersOnly, tt.want.ContainersOnly)
			}
		})
	}
}

func TestSelectorOverwriteViaRebuild(t *testing.T) {
	cfg := NewEmptyOBIConfig()

	// Add with old ports
	oldService := makeServiceSettingWith("my-api", "system", []uint16{3000})
	oldSel := buildOBISelector(oldService, discovery.LangNode)
	if _, err := cfg.AddSelector(oldSel); err != nil {
		t.Fatalf("add old: %v", err)
	}

	// Overwrite with new ports
	newService := makeServiceSettingWith("my-api", "system", []uint16{4000, 4001})
	newSel := buildOBISelector(newService, discovery.LangNode)
	overwritten, err := cfg.AddSelector(newSel)
	if err != nil {
		t.Fatalf("add new: %v", err)
	}
	if !overwritten {
		t.Error("expected overwritten=true")
	}

	// Write and read back
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := cfg.Write(path); err != nil {
		t.Fatalf("Write: %v", err)
	}

	reloaded, err := ReadOBIConfig(path)
	if err != nil {
		t.Fatalf("ReadOBIConfig: %v", err)
	}

	sels := reloaded.ListSelectors()
	if len(sels) != 1 {
		t.Fatalf("expected 1 selector, got %d", len(sels))
	}
	if sels[0].OpenPorts != "4000,4001" {
		t.Errorf("open_ports = %q, want %q", sels[0].OpenPorts, "4000,4001")
	}
}
