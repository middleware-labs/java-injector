//go:build integration

package otelinject

import (
	"testing"
	"time"

	"github.com/middleware-labs/mw-injector/pkg/discovery"
)

func TestE2E_Registry_SystemdRouting(t *testing.T) {
	registry := NewStrategyRegistry()
	svc := discovery.ServiceSetting{
		ServiceName: "test-java",
		SystemdUnit: "test-java",
		Language:    "java",
	}

	strategy := registry.ForService(svc)
	if strategy == nil {
		t.Fatal("expected a strategy for systemd service, got nil")
	}
	if strategy.Name() != "systemd-dropin" {
		t.Errorf("expected systemd-dropin strategy, got %q", strategy.Name())
	}
}

func TestE2E_Registry_OBIRouting(t *testing.T) {
	registry := NewStrategyRegistry()
	svc := discovery.ServiceSetting{
		ServiceName:         "some-go-service",
		Language:            "go",
		InstrumentationType: "obi",
	}

	strategy := registry.ForService(svc)
	if strategy == nil {
		t.Fatal("expected OBI strategy, got nil")
	}
	if strategy.Name() != "obi" {
		t.Errorf("expected obi strategy, got %q", strategy.Name())
	}
}

func TestE2E_Registry_PreferSystemdOverOBI(t *testing.T) {
	registry := NewStrategyRegistry()

	svc := discovery.ServiceSetting{
		ServiceName:         "test-java",
		SystemdUnit:         "test-java",
		Language:            "java",
		InstrumentationType: "obi",
	}

	strategy := registry.ForService(svc)
	if strategy == nil {
		t.Fatal("expected a strategy, got nil")
	}
	if strategy.Name() != "systemd-dropin" {
		t.Errorf("expected systemd-dropin to win over obi (registered first), got %q", strategy.Name())
	}
}

func TestE2E_Registry_InstrumentService_Systemd(t *testing.T) {
	unit := "test-node"
	t.Cleanup(func() { cleanupDropin(t, unit) })

	registry := NewStrategyRegistry()
	svc := discovery.ServiceSetting{
		ServiceName: unit,
		SystemdUnit: unit,
		Language:    "node",
	}

	if err := registry.InstrumentService(svc, discovery.LangNode); err != nil {
		t.Fatalf("InstrumentService failed: %v", err)
	}
	waitForService(t, unit, 10*time.Second)

	if !dropinExists(unit) {
		t.Error("expected drop-in file to exist after InstrumentService")
	}
}

func TestE2E_Registry_OBIFallback(t *testing.T) {
	registry := NewStrategyRegistry()

	languages := []struct {
		lang discovery.Language
		name string
	}{
		{discovery.LangGo, "go"},
		{discovery.LangRust, "rust"},
		{discovery.LangRuby, "ruby"},
		{discovery.LangPHP, "php"},
	}

	for _, tc := range languages {
		t.Run(tc.name, func(t *testing.T) {
			svc := discovery.ServiceSetting{
				ServiceName:         "some-" + tc.name + "-service",
				Language:            string(tc.lang),
				InstrumentationType: "obi",
			}

			strategy := registry.ForService(svc)
			if strategy == nil {
				t.Fatalf("expected OBI strategy for %s, got nil", tc.name)
			}
			if strategy.Name() != "obi" {
				t.Errorf("expected obi strategy for %s, got %q", tc.name, strategy.Name())
			}
		})
	}
}
