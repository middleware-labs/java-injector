package otelinject

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/middleware-labs/java-injector/pkg/discovery"
)

type PythonAgentStatus struct {
	Ready                     bool     `json:"ready"`
	InjectorSharedObjectFound bool     `json:"libotelinject.so_found"`
	LibcFlavor                string   `json:"system_libc_flavor"`
	TargetDirFound            bool     `json:"target_flavor_dir_found"`
	SiteCustomizeFound        bool     `json:"sitecustomize_found"`
	Errors                    []string `json:"errors,omitempty"`
}

type PythonSystemdInjector struct {
	PythonProcs []*discovery.Process
	Status      PythonAgentStatus
	base        baseSystemdInjector
}

func NewPythonSystemdInjector() (*PythonSystemdInjector, error) {
	return NewPythonSystemdInjectorWithLogger(nil)
}

func NewPythonSystemdInjectorWithLogger(logger *slog.Logger) (*PythonSystemdInjector, error) {
	ctx := context.Background()
	pythonProcs, err := discovery.FindProcessesByLanguageWithLogger(ctx, discovery.LangPython, logger)
	if err != nil {
		return nil, fmt.Errorf("error creating PythonSystemdInjector: %w", err)
	}

	ret := &PythonSystemdInjector{
		PythonProcs: pythonProcs,
	}
	ret.base = baseSystemdInjector{
		langName:    "python",
		procs:       ret.PythonProcs,
		applyDropIn: (*SystemdDropin).applySystemdDropInPython,
		dedupByUnit: true,
		ready:       func() bool { return ret.Status.Ready },
	}
	ret.ValidateAssets("")
	return ret, nil
}

func (p *PythonSystemdInjector) ValidateAssets(baseDir string) bool {
	p.Status = ValidatePythonAgent(baseDir)
	return p.Status.Ready
}

func (p *PythonSystemdInjector) Instrument() error         { return p.base.instrument() }
func (p *PythonSystemdInjector) Uninstrument() error       { return p.base.uninstrument() }
func (p *PythonSystemdInjector) InstrumentService(s discovery.ServiceSetting) error {
	return p.base.instrumentService(s)
}
