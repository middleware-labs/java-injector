package otelinject

import (
	"testing"

	"github.com/middleware-labs/mw-injector/pkg/discovery"
)

func TestBuildOBISelector(t *testing.T) {
	tests := []struct {
		name     string
		service  discovery.ServiceSetting
		lang     discovery.Language
		wantSel  OBISelector
	}{
		{
			name: "Java with JAR file",
			service: discovery.ServiceSetting{
				ServiceName: "billing-api",
				ServiceType: "system",
				JarFile:     "billing-api-1.2.3.jar",
				Listeners:   []discovery.Listener{{Port: 8080}},
			},
			lang: discovery.LangJava,
			wantSel: OBISelector{
				Name:      "billing-api",
				Languages: "java",
				OpenPorts: "8080",
				CmdArgs:   "*billing-api-1.2.3.jar*",
			},
		},
		{
			name: "Java with main class no JAR",
			service: discovery.ServiceSetting{
				ServiceName: "billing",
				ServiceType: "system",
				MainClass:   "com.example.BillingService",
			},
			lang: discovery.LangJava,
			wantSel: OBISelector{
				Name:      "billing",
				Languages: "java",
				CmdArgs:   "*com.example.BillingService*",
			},
		},
		{
			name: "Java with neither JAR nor main class",
			service: discovery.ServiceSetting{
				ServiceName: "billing",
				ServiceType: "system",
			},
			lang: discovery.LangJava,
			wantSel: OBISelector{
				Name:      "billing",
				Languages: "java",
			},
		},
		{
			name: "Node uses nodejs semconv name",
			service: discovery.ServiceSetting{
				ServiceName: "my-api",
				ServiceType: "system",
				Listeners:   []discovery.Listener{{Port: 3000}},
			},
			lang: discovery.LangNode,
			wantSel: OBISelector{
				Name:      "my-api",
				Languages: "nodejs",
				OpenPorts: "3000",
			},
		},
		{
			name: "Python uses python semconv name",
			service: discovery.ServiceSetting{
				ServiceName: "flask-app",
				ServiceType: "systemd",
			},
			lang: discovery.LangPython,
			wantSel: OBISelector{
				Name:      "flask-app",
				Languages: "python",
			},
		},
		{
			name: "Rust uses rust semconv name",
			service: discovery.ServiceSetting{
				ServiceName: "my-server",
				ServiceType: "system",
			},
			lang: discovery.LangRust,
			wantSel: OBISelector{
				Name:      "my-server",
				Languages: "rust",
			},
		},
		{
			name: "Go uses go semconv name",
			service: discovery.ServiceSetting{
				ServiceName: "my-gateway",
				ServiceType: "standalone",
			},
			lang: discovery.LangGo,
			wantSel: OBISelector{
				Name:      "my-gateway",
				Languages: "go",
			},
		},
		{
			name: "container process sets ContainersOnly",
			service: discovery.ServiceSetting{
				ServiceName: "billing",
				ServiceType: "docker",
				Listeners:   []discovery.Listener{{Port: 8080}},
			},
			lang: discovery.LangJava,
			wantSel: OBISelector{
				Name:           "billing",
				Languages:      "java",
				OpenPorts:      "8080",
				ContainersOnly: true,
			},
		},
		{
			name: "standalone process does not set ContainersOnly",
			service: discovery.ServiceSetting{
				ServiceName: "billing",
				ServiceType: "standalone",
			},
			lang: discovery.LangJava,
			wantSel: OBISelector{
				Name:      "billing",
				Languages: "java",
			},
		},
		{
			name: "systemd process does not set ContainersOnly",
			service: discovery.ServiceSetting{
				ServiceName: "billing",
				ServiceType: "systemd",
			},
			lang: discovery.LangJava,
			wantSel: OBISelector{
				Name:      "billing",
				Languages: "java",
			},
		},
		{
			name: "system process does not set ContainersOnly",
			service: discovery.ServiceSetting{
				ServiceName: "billing",
				ServiceType: "system",
			},
			lang: discovery.LangPython,
			wantSel: OBISelector{
				Name:      "billing",
				Languages: "python",
			},
		},
		{
			name: "multiple ports comma separated",
			service: discovery.ServiceSetting{
				ServiceName: "billing",
				ServiceType: "system",
				Listeners:   []discovery.Listener{{Port: 8080}, {Port: 9090}, {Port: 443}},
			},
			lang: discovery.LangNode,
			wantSel: OBISelector{
				Name:      "billing",
				Languages: "nodejs",
				OpenPorts: "8080,9090,443",
			},
		},
		{
			name: "no ports results in empty OpenPorts",
			service: discovery.ServiceSetting{
				ServiceName: "billing",
				ServiceType: "system",
			},
			lang: discovery.LangJava,
			wantSel: OBISelector{
				Name:      "billing",
				Languages: "java",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildOBISelector(tt.service, tt.lang)

			if got.Name != tt.wantSel.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantSel.Name)
			}
			if got.Languages != tt.wantSel.Languages {
				t.Errorf("Languages = %q, want %q", got.Languages, tt.wantSel.Languages)
			}
			if got.OpenPorts != tt.wantSel.OpenPorts {
				t.Errorf("OpenPorts = %q, want %q", got.OpenPorts, tt.wantSel.OpenPorts)
			}
			if got.CmdArgs != tt.wantSel.CmdArgs {
				t.Errorf("CmdArgs = %q, want %q", got.CmdArgs, tt.wantSel.CmdArgs)
			}
			if got.ContainersOnly != tt.wantSel.ContainersOnly {
				t.Errorf("ContainersOnly = %v, want %v", got.ContainersOnly, tt.wantSel.ContainersOnly)
			}
		})
	}
}
