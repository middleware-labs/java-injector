package otelinject

import (
	"errors"
	"testing"

	"github.com/middleware-labs/java-injector/pkg/discovery"
)

type mockStrategy struct {
	name          string
	canHandle     func(discovery.ServiceSetting) bool
	validateErr   error
	instrumentErr error
	calls         []string
}

func (m *mockStrategy) Name() string { return m.name }
func (m *mockStrategy) CanHandle(s discovery.ServiceSetting) bool {
	if m.canHandle != nil {
		return m.canHandle(s)
	}
	return false
}
func (m *mockStrategy) ValidateAssets(_ discovery.Language, _ string) error {
	m.calls = append(m.calls, "ValidateAssets")
	return m.validateErr
}
func (m *mockStrategy) Instrument(s discovery.ServiceSetting, lang discovery.Language) error {
	m.calls = append(m.calls, "Instrument")
	return m.instrumentErr
}
func (m *mockStrategy) Uninstrument(s discovery.ServiceSetting) error {
	m.calls = append(m.calls, "Uninstrument")
	return nil
}

func TestStrategyRegistryRouting(t *testing.T) {
	mockSystemd := &mockStrategy{
		name:      "mock-systemd",
		canHandle: func(s discovery.ServiceSetting) bool { return s.SystemdUnit != "" },
	}
	mockOBI := &mockStrategy{
		name:      "mock-obi",
		canHandle: func(s discovery.ServiceSetting) bool { return s.InstrumentationType == "obi" },
	}

	registry := &StrategyRegistry{}
	registry.Register(mockSystemd)
	registry.Register(mockOBI)

	tests := []struct {
		name     string
		service  discovery.ServiceSetting
		wantName string // "" means nil
	}{
		{
			name:     "systemd unit selects first strategy",
			service:  discovery.ServiceSetting{SystemdUnit: "flask.service"},
			wantName: "mock-systemd",
		},
		{
			name:     "obi type without unit selects second strategy",
			service:  discovery.ServiceSetting{InstrumentationType: "obi"},
			wantName: "mock-obi",
		},
		{
			name:     "both match first registered wins",
			service:  discovery.ServiceSetting{SystemdUnit: "flask.service", InstrumentationType: "obi"},
			wantName: "mock-systemd",
		},
		{
			name:     "neither matches returns nil",
			service:  discovery.ServiceSetting{ServiceType: "docker"},
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := registry.ForService(tt.service)
			if tt.wantName == "" {
				if got != nil {
					t.Errorf("expected nil, got strategy %q", got.Name())
				}
				return
			}
			if got == nil {
				t.Fatalf("expected strategy %q, got nil", tt.wantName)
			}
			if got.Name() != tt.wantName {
				t.Errorf("strategy name = %q, want %q", got.Name(), tt.wantName)
			}
		})
	}

	t.Run("empty registry returns nil", func(t *testing.T) {
		empty := &StrategyRegistry{}
		got := empty.ForService(discovery.ServiceSetting{SystemdUnit: "any.service"})
		if got != nil {
			t.Errorf("expected nil from empty registry, got %q", got.Name())
		}
	})
}

func TestInstrumentServiceFlow(t *testing.T) {
	t.Run("validate then instrument called in order", func(t *testing.T) {
		mock := &mockStrategy{
			name:      "test",
			canHandle: func(s discovery.ServiceSetting) bool { return true },
		}
		registry := &StrategyRegistry{}
		registry.Register(mock)

		service := discovery.ServiceSetting{ServiceName: "my-app"}
		err := registry.InstrumentService(service, discovery.LangJava)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(mock.calls) != 2 {
			t.Fatalf("expected 2 calls, got %d: %v", len(mock.calls), mock.calls)
		}
		if mock.calls[0] != "ValidateAssets" {
			t.Errorf("call[0] = %q, want ValidateAssets", mock.calls[0])
		}
		if mock.calls[1] != "Instrument" {
			t.Errorf("call[1] = %q, want Instrument", mock.calls[1])
		}
	})

	t.Run("validate failure skips instrument", func(t *testing.T) {
		mock := &mockStrategy{
			name:        "test",
			canHandle:   func(s discovery.ServiceSetting) bool { return true },
			validateErr: errors.New("agent not found"),
		}
		registry := &StrategyRegistry{}
		registry.Register(mock)

		err := registry.InstrumentService(discovery.ServiceSetting{ServiceName: "my-app"}, discovery.LangJava)
		if err == nil {
			t.Fatal("expected error when validation fails")
		}

		if len(mock.calls) != 1 || mock.calls[0] != "ValidateAssets" {
			t.Errorf("expected only ValidateAssets call, got %v", mock.calls)
		}
	})

	t.Run("no matching strategy returns error", func(t *testing.T) {
		registry := &StrategyRegistry{}
		err := registry.InstrumentService(discovery.ServiceSetting{ServiceName: "my-app"}, discovery.LangJava)
		if err == nil {
			t.Fatal("expected error for no matching strategy")
		}
	})
}

func TestUninstrumentServiceFlow(t *testing.T) {
	t.Run("uninstrument called on matching strategy", func(t *testing.T) {
		mock := &mockStrategy{
			name:      "test",
			canHandle: func(s discovery.ServiceSetting) bool { return true },
		}
		registry := &StrategyRegistry{}
		registry.Register(mock)

		err := registry.UninstrumentService(discovery.ServiceSetting{ServiceName: "my-app"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(mock.calls) != 1 || mock.calls[0] != "Uninstrument" {
			t.Errorf("expected [Uninstrument], got %v", mock.calls)
		}
	})

	t.Run("no matching strategy returns error", func(t *testing.T) {
		registry := &StrategyRegistry{}
		err := registry.UninstrumentService(discovery.ServiceSetting{ServiceName: "my-app"})
		if err == nil {
			t.Fatal("expected error for no matching strategy")
		}
	})
}

func TestDefaultRegistrySystemdSelection(t *testing.T) {
	registry := NewStrategyRegistry()

	t.Run("java with systemd unit selects systemd-dropin", func(t *testing.T) {
		service := discovery.ServiceSetting{
			ServiceName: "billing-api",
			Language:    "java",
			SystemdUnit: "billing-api.service",
		}
		got := registry.ForService(service)
		if got == nil {
			t.Fatal("expected non-nil strategy for systemd java service")
		}
		if got.Name() != "systemd-dropin" {
			t.Errorf("strategy = %q, want %q", got.Name(), "systemd-dropin")
		}
	})

	t.Run("service without unit and non-obi returns nil", func(t *testing.T) {
		service := discovery.ServiceSetting{
			ServiceName: "docker-app",
			Language:    "java",
			ServiceType: "docker",
		}
		got := registry.ForService(service)
		if got != nil {
			t.Errorf("expected nil for docker service without OBI, got %q", got.Name())
		}
	})
}
