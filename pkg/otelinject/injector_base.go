package otelinject

import (
	"errors"
	"fmt"

	"github.com/middleware-labs/java-injector/pkg/discovery"
)

type applyFunc func(*SystemdDropin) error

// baseSystemdInjector holds the shared instrumentation logic for all
// language-specific systemd injectors.
type baseSystemdInjector struct {
	langName    string
	procs       []*discovery.Process
	applyDropIn applyFunc
	dedupByUnit bool
	ready       func() bool
}

func (b *baseSystemdInjector) instrument() error {
	if !b.ready() {
		return fmt.Errorf("%s agent not found", b.langName)
	}

	var errs error
	processedUnits := make(map[string]bool)
	for _, proc := range b.procs {
		isSystemd, unitName := discovery.CheckSystemdStatus(proc.PID)
		if !isSystemd {
			continue
		}
		if b.dedupByUnit && processedUnits[unitName] {
			continue
		}
		if b.dedupByUnit {
			processedUnits[unitName] = true
		}

		dropIn, err := NewSystemdDropin(unitName)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("create dropin for %s (pid %d): %w", unitName, proc.PID, err))
			continue
		}
		if err := b.applyDropIn(dropIn); err != nil {
			errs = errors.Join(errs, fmt.Errorf("apply dropin for %s (pid %d): %w", unitName, proc.PID, err))
		}
	}
	return errs
}

func (b *baseSystemdInjector) uninstrument() error {
	var errs error
	processedUnits := make(map[string]bool)
	for _, proc := range b.procs {
		isSystemd, unitName := discovery.CheckSystemdStatus(proc.PID)
		if !isSystemd {
			continue
		}
		if b.dedupByUnit && processedUnits[unitName] {
			continue
		}
		if b.dedupByUnit {
			processedUnits[unitName] = true
		}
		if err := removeSystemdDropIn(unitName); err != nil {
			errs = errors.Join(errs, fmt.Errorf("remove dropin for %s (pid %d): %w", unitName, proc.PID, err))
		}
	}
	return errs
}

func (b *baseSystemdInjector) instrumentService(service discovery.ServiceSetting) error {
	proc := b.getProcByPID(service.PID)
	if proc == nil {
		return fmt.Errorf("could not find %s process with pid %d on the host", b.langName, service.PID)
	}
	isSystemd, unitName := discovery.CheckSystemdStatus(proc.PID)
	if !isSystemd {
		return fmt.Errorf("%s process with pid %d is not a systemd process", b.langName, service.PID)
	}
	dropIn, err := NewSystemdDropin(unitName)
	if err != nil {
		return fmt.Errorf("create dropin for %s (pid %d): %w", unitName, service.PID, err)
	}
	return b.applyDropIn(dropIn)
}

func (b *baseSystemdInjector) getProcByPID(pid int32) *discovery.Process {
	for _, proc := range b.procs {
		if proc.PID == pid {
			return proc
		}
	}
	return nil
}
