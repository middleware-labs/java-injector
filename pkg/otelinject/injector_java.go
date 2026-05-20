package otelinject

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/middleware-labs/java-injector/pkg/discovery"
)

type JavaAgentStatus struct {
	Ready                     bool     `json:"ready"`
	InjectorSharedObjectFound bool     `json:"libotelinject.so_found"`
	JavaAgentJarFound         bool     `json:"javaagent.jar_found"`
	Errors                    []string `json:"errors,omitempty"`
}

type JavaSystemdInjector struct {
	JavaProcs []*discovery.Process
	Status    JavaAgentStatus
	base      baseSystemdInjector
}

func NewJavaSystemdInjector() (*JavaSystemdInjector, error) {
	return NewJavaSystemdInjectorWithLogger(nil)
}

func NewJavaSystemdInjectorWithLogger(logger *slog.Logger) (*JavaSystemdInjector, error) {
	ctx := context.Background()
	javaProcs, err := discovery.FindProcessesByLanguageWithLogger(ctx, discovery.LangJava, logger)
	if err != nil {
		return nil, fmt.Errorf("error creating JavaSystemdInjector: %w", err)
	}

	ret := &JavaSystemdInjector{
		JavaProcs: javaProcs,
	}
	ret.base = baseSystemdInjector{
		langName:    "java",
		procs:       ret.JavaProcs,
		applyDropIn: (*SystemdDropin).applySystemdDropIn,
		ready:       func() bool { return ret.Status.Ready },
	}
	ret.ValidateAssets("")
	return ret, nil
}

func (j *JavaSystemdInjector) ValidateAssets(baseDir string) bool {
	j.Status = ValidateJavaAgent(baseDir)
	return j.Status.Ready
}

func (j *JavaSystemdInjector) Instrument() error         { return j.base.instrument() }
func (j *JavaSystemdInjector) Uninstrument() error       { return j.base.uninstrument() }
func (j *JavaSystemdInjector) InstrumentService(s discovery.ServiceSetting) error {
	return j.base.instrumentService(s)
}
