package otelinject

import (
	"fmt"

	"github.com/middleware-labs/mw-injector/pkg/discovery"
)

// SystemdDropinStrategy instruments processes by creating systemd drop-in
// files that set OTel environment variables (LD_PRELOAD, OTEL_SERVICE_NAME,
// etc.). The drop-in is placed at:
//
//	/etc/systemd/system/{unit}.service.d/middleware-otel.conf
//
// After writing, systemd is reloaded and the service restarted.
//
// Language differences:
//   - Java/Node: LD_PRELOAD with libotelinject.so (applySystemdDropIn)
//   - Python: LD_PRELOAD + PYTHON_AUTO_INSTRUMENTATION_AGENT_PATH_PREFIX
//     and HTTP-based exporters (applySystemdDropInPython)
type SystemdDropinStrategy struct{}

// Name returns "systemd-dropin".
func (s *SystemdDropinStrategy) Name() string {
	return "systemd-dropin"
}

// CanHandle returns true if the service has a systemd unit and is a language
// that supports LD_PRELOAD-based injection (Java, Node, Python). Go binaries
// are statically linked and don't support LD_PRELOAD — they require OBI.
func (s *SystemdDropinStrategy) CanHandle(service discovery.ServiceSetting) bool {
	if service.SystemdUnit == "" {
		return false
	}
	meta, ok := languageMeta[discovery.Language(service.Language)]
	return ok && meta.SupportsSystemdDropin
}

// ValidateAssets checks that required agent files are present for the given
// language. Each language has its own asset layout:
//   - Java: javaagent.jar + libotelinject.so
//   - Node: register.js + node_modules + libotelinject.so
//   - Python: sitecustomize.py + opentelemetry libs + libotelinject.so
func (s *SystemdDropinStrategy) ValidateAssets(lang discovery.Language, baseDir string) error {
	meta, ok := languageMeta[lang]
	if !ok || !meta.SupportsSystemdDropin || meta.ValidateAgent == nil {
		return fmt.Errorf("unsupported language: %s", lang)
	}
	return meta.ValidateAgent(baseDir)
}

// Instrument creates a systemd drop-in file for the service's unit and
// restarts it. The drop-in content varies by language — Python requires
// additional environment variables for the OTel Python auto-instrumentation.
func (s *SystemdDropinStrategy) Instrument(service discovery.ServiceSetting, lang discovery.Language) error {
	unitName := service.SystemdUnit
	if unitName == "" {
		return fmt.Errorf("service %q has no systemd unit", service.ServiceName)
	}

	meta, ok := languageMeta[lang]
	if !ok || !meta.SupportsSystemdDropin || meta.InstrumentDropin == nil {
		return fmt.Errorf("unsupported language for systemd drop-in: %s", lang)
	}

	dropIn, err := NewSystemdDropin(unitName)
	if err != nil {
		return fmt.Errorf("failed to create systemd dropin for %s: %w", unitName, err)
	}

	return meta.InstrumentDropin(dropIn)
}

// Uninstrument removes the systemd drop-in file for the service's unit
// and restarts it. This is language-agnostic — the same drop-in file
// is removed regardless of language.
func (s *SystemdDropinStrategy) Uninstrument(service discovery.ServiceSetting) error {
	unitName := service.SystemdUnit
	if unitName == "" {
		return fmt.Errorf("service %q has no systemd unit", service.ServiceName)
	}

	return removeSystemdDropIn(unitName)
}
