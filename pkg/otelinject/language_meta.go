package otelinject

import (
	"fmt"

	"github.com/middleware-labs/mw-injector/pkg/discovery"
)

// LanguageMeta holds per-language configuration for instrumentation strategies.
// Adding a new language requires adding one entry to this table.
type LanguageMeta struct {
	// OBISemconvName is the OTel semantic conventions language name used in
	// OBI selectors (e.g. "nodejs" for Node, "java" for Java).
	OBISemconvName string

	// SupportsSystemdDropin is true when the language can be instrumented
	// via LD_PRELOAD-based systemd drop-in files.
	SupportsSystemdDropin bool

	// ValidateAgent checks that required agent assets exist for systemd
	// drop-in instrumentation. Only meaningful when SupportsSystemdDropin is true.
	ValidateAgent func(baseDir string) error

	// InstrumentDropin applies the language-specific systemd drop-in. Python
	// needs extra env vars; Java and Node use the same LD_PRELOAD drop-in.
	InstrumentDropin func(dropIn *SystemdDropin) error

	// CustomizeOBISelector allows a language to add extra fields to the OBI
	// selector (e.g. Java adds CmdArgs for jar/class disambiguation).
	CustomizeOBISelector func(sel *OBISelector, svc discovery.ServiceSetting)
}

var languageMeta = map[discovery.Language]LanguageMeta{
	discovery.LangJava: {
		OBISemconvName:        "java",
		SupportsSystemdDropin: true,
		ValidateAgent: func(baseDir string) error {
			if s := ValidateJavaAgent(baseDir); !s.Ready {
				return fmt.Errorf("java agent not ready: %v", s.Errors)
			}
			return nil
		},
		InstrumentDropin: func(d *SystemdDropin) error { return d.applySystemdDropIn() },
		CustomizeOBISelector: func(sel *OBISelector, svc discovery.ServiceSetting) {
			if svc.JarFile != "" {
				sel.CmdArgs = "*" + svc.JarFile + "*"
			} else if svc.MainClass != "" {
				sel.CmdArgs = "*" + svc.MainClass + "*"
			}
		},
	},
	discovery.LangNode: {
		OBISemconvName:        "nodejs",
		SupportsSystemdDropin: true,
		ValidateAgent: func(baseDir string) error {
			if s := ValidateNodeAgent(baseDir); !s.Ready {
				return fmt.Errorf("node agent not ready: %v", s.Errors)
			}
			return nil
		},
		InstrumentDropin: func(d *SystemdDropin) error { return d.applySystemdDropIn() },
	},
	discovery.LangPython: {
		OBISemconvName:        "python",
		SupportsSystemdDropin: true,
		ValidateAgent: func(baseDir string) error {
			if s := ValidatePythonAgent(baseDir); !s.Ready {
				return fmt.Errorf("python agent not ready: %v", s.Errors)
			}
			return nil
		},
		InstrumentDropin: func(d *SystemdDropin) error { return d.applySystemdDropInPython() },
	},
	discovery.LangGo:   {OBISemconvName: "go"},
	discovery.LangRust:  {OBISemconvName: "rust"},
	discovery.LangRuby:  {OBISemconvName: "ruby"},
	discovery.LangPHP:   {OBISemconvName: "php"},
}
