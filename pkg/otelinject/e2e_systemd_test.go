//go:build integration

package otelinject

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/middleware-labs/java-injector/pkg/discovery"
)

func TestMain(m *testing.M) {
	pid1, _ := os.Readlink("/proc/1/exe")
	if !strings.Contains(pid1, "systemd") {
		fmt.Fprintln(os.Stderr, "SKIP: integration tests require systemd as PID 1")
		os.Exit(0)
	}

	for _, svc := range []string{"test-java", "test-node", "test-python", "obi-agent"} {
		if exec.Command("systemctl", "is-active", "--quiet", svc).Run() != nil {
			fmt.Fprintf(os.Stderr, "SKIP: required service %s is not active\n", svc)
			os.Exit(0)
		}
	}

	os.Exit(m.Run())
}

// --- helpers ---

func waitForService(t *testing.T, unit string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if exec.Command("systemctl", "is-active", "--quiet", unit).Run() == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("service %s did not become active within %v", unit, timeout)
}

func getServicePID(t *testing.T, unit string) string {
	t.Helper()
	out, err := exec.Command("systemctl", "show", "-p", "MainPID", "--value", unit).Output()
	if err != nil {
		t.Fatalf("failed to get MainPID for %s: %v", unit, err)
	}
	return strings.TrimSpace(string(out))
}

func getServiceEnvironment(t *testing.T, unit string) string {
	t.Helper()
	out, err := exec.Command("systemctl", "show", "-p", "Environment", "--value", unit).Output()
	if err != nil {
		t.Fatalf("failed to get Environment for %s: %v", unit, err)
	}
	return strings.TrimSpace(string(out))
}

func readDropinFile(t *testing.T, unit string) string {
	t.Helper()
	path := fmt.Sprintf("/etc/systemd/system/%s.service.d/middleware-otel.conf", unit)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read drop-in file %s: %v", path, err)
	}
	return string(data)
}

func dropinExists(unit string) bool {
	path := fmt.Sprintf("/etc/systemd/system/%s.service.d/middleware-otel.conf", unit)
	_, err := os.Stat(path)
	return err == nil
}

func dropinDirExists(unit string) bool {
	dir := fmt.Sprintf("/etc/systemd/system/%s.service.d", unit)
	_, err := os.Stat(dir)
	return err == nil
}

func cleanupDropin(t *testing.T, unit string) {
	t.Helper()
	_ = UninstrumentUnit(unit)
	_ = exec.Command("systemctl", "daemon-reload").Run()
	waitForService(t, unit, 10*time.Second)
}

// --- systemd drop-in tests ---

func TestE2E_InstrumentUnit_Java(t *testing.T) {
	unit := "test-java"
	t.Cleanup(func() { cleanupDropin(t, unit) })

	if err := InstrumentUnit(unit, discovery.LangJava); err != nil {
		t.Fatalf("InstrumentUnit(%s, Java) failed: %v", unit, err)
	}
	waitForService(t, unit, 10*time.Second)

	content := readDropinFile(t, unit)

	expected := []string{
		`LD_PRELOAD=/usr/lib/opentelemetry/libotelinject.so`,
		`OTEL_SERVICE_NAME=test-java`,
		`OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:9319`,
		`OTEL_EXPORTER_OTLP_PROTOCOL=grpc`,
	}
	for _, want := range expected {
		if !strings.Contains(content, want) {
			t.Errorf("drop-in missing %q\ngot:\n%s", want, content)
		}
	}

	env := getServiceEnvironment(t, unit)
	for _, want := range expected {
		if !strings.Contains(env, want) {
			t.Errorf("systemctl show Environment missing %q\ngot: %s", want, env)
		}
	}
}

func TestE2E_InstrumentUnit_Node(t *testing.T) {
	unit := "test-node"
	t.Cleanup(func() { cleanupDropin(t, unit) })

	if err := InstrumentUnit(unit, discovery.LangNode); err != nil {
		t.Fatalf("InstrumentUnit(%s, Node) failed: %v", unit, err)
	}
	waitForService(t, unit, 10*time.Second)

	content := readDropinFile(t, unit)

	expected := []string{
		`LD_PRELOAD=/usr/lib/opentelemetry/libotelinject.so`,
		`OTEL_SERVICE_NAME=test-node`,
		`OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:9319`,
		`OTEL_EXPORTER_OTLP_PROTOCOL=grpc`,
	}
	for _, want := range expected {
		if !strings.Contains(content, want) {
			t.Errorf("drop-in missing %q\ngot:\n%s", want, content)
		}
	}
}

func TestE2E_InstrumentUnit_Python(t *testing.T) {
	unit := "test-python"
	t.Cleanup(func() { cleanupDropin(t, unit) })

	if err := InstrumentUnit(unit, discovery.LangPython); err != nil {
		t.Fatalf("InstrumentUnit(%s, Python) failed: %v", unit, err)
	}
	waitForService(t, unit, 10*time.Second)

	content := readDropinFile(t, unit)

	pythonExpected := []string{
		`LD_PRELOAD=/usr/lib/opentelemetry/libotelinject.so`,
		`OTEL_SERVICE_NAME=test-python`,
		`PYTHON_AUTO_INSTRUMENTATION_AGENT_PATH_PREFIX=/opt/otel-python-agent`,
		`OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:9319`,
		`OTEL_EXPORTER_OTLP_PROTOCOL=grpc`,
		`OTEL_TRACES_EXPORTER=otlp_proto_http`,
		`OTEL_METRICS_EXPORTER=otlp_proto_http`,
		`OTEL_LOGS_EXPORTER=otlp_proto_http`,
	}
	for _, want := range pythonExpected {
		if !strings.Contains(content, want) {
			t.Errorf("drop-in missing %q\ngot:\n%s", want, content)
		}
	}

	env := getServiceEnvironment(t, unit)
	if !strings.Contains(env, "PYTHON_AUTO_INSTRUMENTATION_AGENT_PATH_PREFIX") {
		t.Errorf("systemctl show missing PYTHON_AUTO_INSTRUMENTATION_AGENT_PATH_PREFIX\ngot: %s", env)
	}
}

func TestE2E_UninstrumentUnit_Cleanup(t *testing.T) {
	unit := "test-java"

	if err := InstrumentUnit(unit, discovery.LangJava); err != nil {
		t.Fatalf("InstrumentUnit failed: %v", err)
	}
	waitForService(t, unit, 10*time.Second)

	if !dropinExists(unit) {
		t.Fatal("drop-in file should exist after instrumentation")
	}

	if err := UninstrumentUnit(unit); err != nil {
		t.Fatalf("UninstrumentUnit failed: %v", err)
	}
	waitForService(t, unit, 10*time.Second)

	if dropinExists(unit) {
		t.Error("drop-in file should be removed after uninstrumentation")
	}
	if dropinDirExists(unit) {
		t.Error("drop-in directory should be removed when empty")
	}

	env := getServiceEnvironment(t, unit)
	if strings.Contains(env, "OTEL_SERVICE_NAME") {
		t.Errorf("environment should be clean after uninstrumentation, got: %s", env)
	}
}

func TestE2E_InstrumentUnit_ServiceRestart(t *testing.T) {
	unit := "test-java"
	t.Cleanup(func() { cleanupDropin(t, unit) })

	pidBefore := getServicePID(t, unit)

	if err := InstrumentUnit(unit, discovery.LangJava); err != nil {
		t.Fatalf("InstrumentUnit failed: %v", err)
	}
	waitForService(t, unit, 10*time.Second)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		pidAfter := getServicePID(t, unit)
		if pidAfter != pidBefore && pidAfter != "0" {
			return // PID changed, restart confirmed
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Error("service PID did not change after instrumentation (restart expected)")
}

func TestE2E_InstrumentUnit_Idempotent(t *testing.T) {
	unit := "test-java"
	t.Cleanup(func() { cleanupDropin(t, unit) })

	if err := InstrumentUnit(unit, discovery.LangJava); err != nil {
		t.Fatalf("first InstrumentUnit failed: %v", err)
	}
	waitForService(t, unit, 10*time.Second)

	if err := InstrumentUnit(unit, discovery.LangJava); err != nil {
		t.Fatalf("second InstrumentUnit failed: %v", err)
	}
	waitForService(t, unit, 10*time.Second)

	content := readDropinFile(t, unit)
	if !strings.Contains(content, "OTEL_SERVICE_NAME=test-java") {
		t.Errorf("drop-in content wrong after second instrument:\n%s", content)
	}
}

func TestE2E_InstrumentUnit_UnsupportedLanguage(t *testing.T) {
	err := InstrumentUnit("test-java", discovery.LangRust)
	if err == nil {
		t.Fatal("expected error for unsupported language Rust, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected 'unsupported' in error, got: %v", err)
	}
}
