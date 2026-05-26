//go:build integration

package otelinject

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/middleware-labs/java-injector/pkg/discovery"
)

const obiConfigPath = "/etc/obi-agent/config.yaml"

func backupOBIConfig(t *testing.T) {
	t.Helper()
	original, err := os.ReadFile(obiConfigPath)
	if err != nil {
		t.Fatalf("failed to backup OBI config: %v", err)
	}
	t.Cleanup(func() {
		os.WriteFile(obiConfigPath, original, 0644)
	})
}

func TestE2E_OBIStrategy_ValidateAssets(t *testing.T) {
	strategy := NewOBIStrategy()
	if err := strategy.ValidateAssets(discovery.LangJava, ""); err != nil {
		t.Fatalf("ValidateAssets failed: %v", err)
	}
}

func TestE2E_OBIStrategy_Instrument(t *testing.T) {
	backupOBIConfig(t)

	strategy := NewOBIStrategyWithLogger(slog.Default())
	setting := discovery.ServiceSetting{
		ServiceName: "e2e-obi-test-svc",
		Language:    "java",
		ServiceType: "system",
	}

	if err := strategy.Instrument(setting, discovery.LangJava); err != nil {
		t.Fatalf("Instrument failed: %v", err)
	}

	cfg, err := ReadOBIConfig(obiConfigPath)
	if err != nil {
		t.Fatalf("ReadOBIConfig after instrument: %v", err)
	}

	if !cfg.HasSelector("e2e-obi-test-svc") {
		t.Error("expected selector 'e2e-obi-test-svc' in OBI config after instrument")
	}

	selectors := cfg.ListSelectors()
	for _, sel := range selectors {
		if sel.Name == "e2e-obi-test-svc" {
			if sel.Languages != "java" {
				t.Errorf("expected languages=java, got %q", sel.Languages)
			}
			return
		}
	}
	t.Error("selector 'e2e-obi-test-svc' not found in ListSelectors")
}

func TestE2E_OBIStrategy_Uninstrument(t *testing.T) {
	backupOBIConfig(t)

	strategy := NewOBIStrategyWithLogger(slog.Default())
	setting := discovery.ServiceSetting{
		ServiceName: "e2e-obi-uninstrument-svc",
		Language:    "java",
		ServiceType: "system",
	}

	if err := strategy.Instrument(setting, discovery.LangJava); err != nil {
		t.Fatalf("Instrument failed: %v", err)
	}

	if err := strategy.Uninstrument(setting); err != nil {
		t.Fatalf("Uninstrument failed: %v", err)
	}

	cfg, err := ReadOBIConfig(obiConfigPath)
	if err != nil {
		t.Fatalf("ReadOBIConfig after uninstrument: %v", err)
	}

	if cfg.HasSelector("e2e-obi-uninstrument-svc") {
		t.Error("selector should be removed after uninstrumentation")
	}
}

func TestE2E_OBIConfig_RoundTrip(t *testing.T) {
	backupOBIConfig(t)

	cfg, err := ReadOBIConfig(obiConfigPath)
	if err != nil {
		t.Fatalf("ReadOBIConfig failed: %v", err)
	}

	sel := OBISelector{
		Name:      "roundtrip-test",
		Languages: "nodejs",
		OpenPorts: "3000",
	}
	if _, err := cfg.AddSelector(sel); err != nil {
		t.Fatalf("AddSelector failed: %v", err)
	}
	if err := cfg.Write(obiConfigPath); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	raw, err := os.ReadFile(obiConfigPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(raw)

	if !strings.Contains(content, "otel_traces_exporter: otlp") {
		t.Error("round-trip lost otel_traces_exporter field")
	}
	if !strings.Contains(content, "target_url: http://localhost:9319") {
		t.Error("round-trip lost target_url field")
	}
	if !strings.Contains(content, "poll_interval: 5s") {
		t.Error("round-trip lost poll_interval field")
	}
	if !strings.Contains(content, "# OBI Agent Configuration") {
		t.Error("round-trip lost top-level comment")
	}
	if !strings.Contains(content, "roundtrip-test") {
		t.Error("round-trip lost the added selector")
	}
}

func TestE2E_OBIStrategy_RestartVerified(t *testing.T) {
	backupOBIConfig(t)

	pidBefore := getServicePID(t, "obi-agent")

	strategy := NewOBIStrategyWithLogger(slog.Default())
	setting := discovery.ServiceSetting{
		ServiceName: "e2e-restart-check",
		Language:    "java",
		ServiceType: "system",
	}

	if err := strategy.Instrument(setting, discovery.LangJava); err != nil {
		t.Fatalf("Instrument failed: %v", err)
	}

	pidAfter := getServicePID(t, "obi-agent")
	if pidAfter == pidBefore {
		t.Error("obi-agent PID did not change after instrument (restart expected)")
	}
}

func TestE2E_InstrumentOBI_FullFlow(t *testing.T) {
	backupOBIConfig(t)

	logger := slog.Default()
	if err := InstrumentOBI("test-java", LanguageJava, logger); err != nil {
		t.Fatalf("InstrumentOBI full flow failed: %v", err)
	}

	cfg, err := ReadOBIConfig(obiConfigPath)
	if err != nil {
		t.Fatalf("ReadOBIConfig: %v", err)
	}

	if !cfg.HasSelector("test-java") {
		t.Error("expected selector 'test-java' in OBI config after InstrumentOBI")
	}

	selectors := cfg.ListSelectors()
	for _, sel := range selectors {
		if sel.Name == "test-java" {
			if sel.Languages != "java" {
				t.Errorf("expected languages=java, got %q", sel.Languages)
			}
			return
		}
	}
	t.Error("selector 'test-java' not found in ListSelectors")
}

func TestE2E_UninstrumentOBI_FullFlow(t *testing.T) {
	backupOBIConfig(t)

	logger := slog.Default()
	if err := InstrumentOBI("test-java", LanguageJava, logger); err != nil {
		t.Fatalf("InstrumentOBI failed: %v", err)
	}

	if err := UninstrumentOBI("test-java", logger); err != nil {
		t.Fatalf("UninstrumentOBI failed: %v", err)
	}

	cfg, err := ReadOBIConfig(obiConfigPath)
	if err != nil {
		t.Fatalf("ReadOBIConfig: %v", err)
	}

	if cfg.HasSelector("test-java") {
		t.Error("selector 'test-java' should be removed after UninstrumentOBI")
	}
}

func TestE2E_InstrumentOBIBulk(t *testing.T) {
	backupOBIConfig(t)

	logger := slog.Default()
	if err := InstrumentOBIBulk(LanguageJava, logger); err != nil {
		t.Fatalf("InstrumentOBIBulk failed: %v", err)
	}

	cfg, err := ReadOBIConfig(obiConfigPath)
	if err != nil {
		t.Fatalf("ReadOBIConfig: %v", err)
	}

	selectors := cfg.ListSelectors()
	found := false
	for _, sel := range selectors {
		if sel.Languages == "java" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one Java selector after InstrumentOBIBulk")
	}
}
