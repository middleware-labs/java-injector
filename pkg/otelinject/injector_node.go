package otelinject

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/middleware-labs/mw-injector/pkg/discovery"
)

type NodeAgentStatus struct {
	Ready                     bool     `json:"ready"`
	InjectorSharedObjectFound bool     `json:"libotelinject.so_found"`
	RegisterJSFound           bool     `json:"register_js_found"`
	PackageVersion            string   `json:"package_version,omitempty"`
	MissingDeps               []string `json:"missing_deps,omitempty"`
	Errors                    []string `json:"errors,omitempty"`
}

type NodeSystemdInjector struct {
	NodeProcs []*discovery.Process
	Status    NodeAgentStatus
	base      baseSystemdInjector
}

func NewNodeSystemdInjector() (*NodeSystemdInjector, error) {
	return NewNodeSystemdInjectorWithLogger(nil)
}

func NewNodeSystemdInjectorWithLogger(logger *slog.Logger) (*NodeSystemdInjector, error) {
	ctx := context.Background()
	nodeProcs, err := discovery.FindProcessesByLanguageWithLogger(ctx, discovery.LangNode, logger)
	if err != nil {
		return nil, fmt.Errorf("error creating NodeSystemdInjector: %w", err)
	}

	ret := &NodeSystemdInjector{
		NodeProcs: nodeProcs,
	}
	ret.base = baseSystemdInjector{
		langName:    "node",
		procs:       ret.NodeProcs,
		applyDropIn: (*SystemdDropin).applySystemdDropIn,
		ready:       func() bool { return ret.Status.Ready },
	}
	ret.ValidateAssets("")
	return ret, nil
}

func (n *NodeSystemdInjector) ValidateAssets(baseDir string) bool {
	n.Status = ValidateNodeAgent(baseDir)
	return n.Status.Ready
}

func (n *NodeSystemdInjector) Instrument() error         { return n.base.instrument() }
func (n *NodeSystemdInjector) Uninstrument() error       { return n.base.uninstrument() }
func (n *NodeSystemdInjector) InstrumentService(s discovery.ServiceSetting) error {
	return n.base.instrumentService(s)
}
