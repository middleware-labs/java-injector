package otelinject

import (
	"context"
	"log/slog"
	"sort"

	"github.com/middleware-labs/mw-injector/pkg/discovery"
)

// IntegrationEntry represents a discovered infrastructure integration
// (Redis, MySQL, Kafka, etc.) with one or more running instances.
type IntegrationEntry struct {
	IntegrationType string
	ServiceName     string
	ServiceType     string // "systemd", "docker", "standalone"
	SystemdUnit     string
	Ports           []int
	Instances       []InstanceInfo
}

// DiscoverIntegrationsOpts controls filtering for DiscoverIntegrations.
type DiscoverIntegrationsOpts struct {
	Logger *slog.Logger
}

// DiscoverIntegrations discovers infrastructure integrations running on the
// host. It combines two sources:
//  1. Non-language processes matched by integration rules (Redis, Nginx, etc.)
//  2. Language-classified processes that are also known integrations (Kafka=Java, etc.)
func DiscoverIntegrations(opts DiscoverIntegrationsOpts) ([]IntegrationEntry, error) {
	ctx := context.Background()

	langProcs, integrationProcs, err := discovery.FindAllWithIntegrations(ctx, opts.Logger)
	if err != nil {
		return nil, err
	}

	type group struct {
		integrationType string
		serviceName     string
		serviceType     string
		systemdUnit     string
		ports           map[int]struct{}
		instances       []InstanceInfo
	}

	groups := make(map[string]*group)
	var order []string

	addProc := func(proc *discovery.Process) {
		itype := proc.IntegrationType()
		if itype == "" {
			return
		}

		key := itype
		g, exists := groups[key]
		if !exists {
			serviceType := "standalone"
			systemdUnit := proc.DetailString(discovery.DetailSystemdUnit)
			if systemdUnit != "" {
				serviceType = "systemd"
			} else if proc.IsInContainer() {
				serviceType = "docker"
			}
			g = &group{
				integrationType: itype,
				serviceName:     proc.ServiceName,
				serviceType:     serviceType,
				systemdUnit:     systemdUnit,
				ports:           make(map[int]struct{}),
			}
			groups[key] = g
			order = append(order, key)
		}

		inst := InstanceInfo{
			PID:    proc.PID,
			Owner:  proc.Owner,
			Status: proc.Status,
		}
		seen := make(map[int]struct{})
		for _, l := range proc.Listeners() {
			p := int(l.Port)
			if _, dup := seen[p]; !dup {
				inst.Ports = append(inst.Ports, p)
				seen[p] = struct{}{}
			}
			g.ports[p] = struct{}{}
		}
		sort.Ints(inst.Ports)
		g.instances = append(g.instances, inst)
	}

	for _, procs := range langProcs {
		for _, proc := range procs {
			addProc(proc)
		}
	}
	for _, proc := range integrationProcs {
		addProc(proc)
	}

	entries := make([]IntegrationEntry, 0, len(groups))
	for _, key := range order {
		g := groups[key]

		ports := make([]int, 0, len(g.ports))
		for p := range g.ports {
			ports = append(ports, p)
		}
		sort.Ints(ports)

		entries = append(entries, IntegrationEntry{
			IntegrationType: g.integrationType,
			ServiceName:     g.serviceName,
			ServiceType:     g.serviceType,
			SystemdUnit:     g.systemdUnit,
			Ports:           ports,
			Instances:       g.instances,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].IntegrationType < entries[j].IntegrationType
	})

	return entries, nil
}
