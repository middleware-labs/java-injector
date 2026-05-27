# Testing Guide

## Overview

The test suite is organized in four layers, each building on the previous:

| Layer | What it tests | I/O | Build tag | Run time |
|-------|--------------|-----|-----------|----------|
| Layer 1 | Pure logic (service names, cgroup parsing, naming) | None | — | <1s |
| Layer 2 | Mocked I/O (cache, container clients, HTTP inspect) | In-memory mocks | — | <1s |
| Layer 3 | Integration with mocks (OBI config, strategy routing) | Temp files | — | <1s |
| Layer 4 | End-to-end (systemd drop-ins, OBI, discovery) | Real systemd in Docker | `integration` | ~50s |

## Running Tests

```bash
# All unit tests (Layers 1-3)
go test ./...

# Verbose, specific package
go test -v ./pkg/discovery
go test -v ./pkg/otelinject
go test -v ./pkg/javanaming

# Specific test
go test -v ./pkg/discovery -run TestCleanName
go test -v ./pkg/otelinject -run TestOBIConfig

# End-to-end integration tests (Layer 4, requires Docker)
bash tests/integration/run.sh
```

## Layer 1 — Pure Logic Tests

No file system, no /proc, no network. Test functions receive data directly.

### pkg/discovery

| File | What it covers | Key test count |
|------|---------------|----------------|
| `service_name_test.go` | `cleanName` (generic filtering, sanitization), `serviceNameFromWorkDir` | 51 cases |
| `java_service_name_test.go` | Java service name priority chain, system properties, JAR version stripping | ~20 cases |
| `node_service_name_test.go` | Node service name extraction from cmdArgs, scripts, package.json | 43 cases |
| `python_service_name_test.go` | Python priority chain, `isGenericPython`, `PythonDetect` | 61 cases |
| `container_cgroup_test.go` | `parseCgroupContent` (Docker/K8s/Podman/LXC), `parseCgroupUnitContent` | 33 cases |
| `container_util_test.go` | `splitImageTag`, `extractComposeInfo`, `runtimeMatchesClient` | 21 cases |

### pkg/javanaming

| File | What it covers | Key test count |
|------|---------------|----------------|
| `naming_test.go` | `CleanServiceName`, `CleanJarName`, `CleanTomcatInstance`, `IsGenericName`, `GenerateForStandard` | 49 cases |

### pkg/otelinject

| File | What it covers | Key test count |
|------|---------------|----------------|
| `obi_selector_test.go` | `buildOBISelector` — language semconv mapping, ContainersOnly, ports, CmdArgs | 13 cases |
| `dropin_test.go` | Drop-in validation, `NewSystemdDropin`, `shellescape` | 20 cases |

## Layer 2 — Mocked I/O Tests

Use in-memory mocks or `httptest` servers. No real files or sockets.

### pkg/discovery

| File | What it covers |
|------|---------------|
| `cache_test.go` | Process cache: put/get, TTL prune, key isolation, ContainerInfo preservation (9 tests) |
| `container_client_test.go` | `mockContainerClient` — batch resolution, dedup, runtime routing, cache (13 subtests) |
| `container_client_http_test.go` | `httptest` server mocking `unixSocketClient` — JSON parsing, error codes, partial failures (9 subtests) |

## Layer 3 — Integration Tests (Mocked Strategies)

Use temp files and mock strategy implementations. No systemd or Docker.

### pkg/otelinject

| File | What it covers |
|------|---------------|
| `obiconfig_integration_test.go` | Multi-operation YAML round-trip, bulk operations, null sequences |
| `obi_selector_integration_test.go` | Selector building, persistence, overwrite |
| `strategy_integration_test.go` | Strategy registry routing, instrument/uninstrument flows |
| `grouping_integration_test.go` | Service entry grouping from settings |
| `container_pipeline_test.go` | Full container resolution pipeline |

## Layer 4 — End-to-End Tests (Systemd in Docker)

These tests run inside a Docker container with **systemd as PID 1**. They exercise real `InstrumentUnit`, `InstrumentOBI`, and `DiscoverServices` calls against actual systemd services.

### Prerequisites

- Docker installed and running
- `--privileged` capability (required for systemd in Docker)

### How to run

```bash
bash tests/integration/run.sh
```

This script:
1. Builds the Docker image (~3 min first time, cached after)
2. Starts a privileged container with systemd as PID 1
3. Waits for systemd boot + all 4 test services to become active
4. Runs `go test -tags integration ./pkg/otelinject/`
5. Cleans up the container

### How fake processes work

The detection handlers identify language by reading `/proc/<pid>/exe`. Symlinks don't work because `readlink` resolves to the target binary. Solution: **copy** `/bin/sleep` to `/usr/local/bin/java`, `/usr/local/bin/node`, `/usr/local/bin/python3`.

```
Dockerfile:  cp /bin/sleep /usr/local/bin/java     # physical copy, own inode
Unit file:   ExecStart=/usr/local/bin/java 3600     # systemd launches it

Kernel:      /proc/<pid>/exe → /usr/local/bin/java  # real file, not symlink
Scanner:     filepath.Base(readlink) → "java"        # ExeName = "java"
Java Detect: exeLower == "java" → true               # detection passes

Cgroup:      0::/system.slice/.../test-java.service  # systemd unit visible
Enrich:      parseCgroupUnitName → "test-java"       # service name extracted
```

The process is functionally `sleep 3600` (stays alive for an hour), but the kernel and discovery code see it as a Java process managed by the `test-java` systemd unit.

### What's inside the container

| Component | Path / Description |
|-----------|-------------------|
| Fake Java binary | `/usr/local/bin/java` (copy of sleep) |
| Fake Node binary | `/usr/local/bin/node` (copy of sleep) |
| Fake Python binary | `/usr/local/bin/python3` (copy of sleep) |
| Java agent stub | `/usr/lib/opentelemetry/jvm/javaagent.jar` (empty) |
| Node agent stubs | `/usr/lib/opentelemetry/nodejs/node_modules/...` (empty dirs/files) |
| Python agent stubs | `/opt/otel-python-agent/glibc/sitecustomize.py` (empty) |
| Shared library stub | `/usr/lib/opentelemetry/libotelinject.so` (empty) |
| OBI binary stub | `/usr/local/bin/obi` (copy of sleep) |
| OBI config | `/etc/obi-agent/config.yaml` (seed with comments) |
| Test services | `test-java`, `test-node`, `test-python`, `obi-agent` systemd units |

### Test files

| File | Tests | What it covers |
|------|-------|---------------|
| `e2e_systemd_test.go` | 8 | `InstrumentUnit`/`UninstrumentUnit` for Java/Node/Python. Verifies drop-in file content, env vars via `systemctl show`, cleanup (file + directory removed), service restart, idempotent re-instrumentation, unsupported language rejection. |
| `e2e_obi_test.go` | 8 | OBI strategy `ValidateAssets`, `Instrument`/`Uninstrument` with config verification, YAML round-trip preservation, obi-agent restart, full-flow `InstrumentOBI`/`UninstrumentOBI`/`InstrumentOBIBulk` using real discovery. |
| `e2e_discovery_test.go` | 6 | `FindAllProcesses` finds fake processes, `ListSystemdServices` returns correct language/unit, `DiscoverServices` grouping, fingerprint stability, `SystemdUnit` population. |
| `e2e_strategy_test.go` | 5 | Registry routes to `SystemdDropinStrategy` vs `OBIStrategy`, systemd wins when both match, `InstrumentService` creates drop-in, Go/Rust/Ruby/PHP fall back to OBI. |

### Test isolation

- **Systemd tests**: Each test cleans up via `t.Cleanup` — removes drop-in file, runs `daemon-reload`, waits for service active.
- **OBI tests**: Back up `/etc/obi-agent/config.yaml` before each test, restore in `t.Cleanup`.
- **TestMain**: Verifies systemd is PID 1 and all services are active. Gracefully skips (not fails) if not in the test container.

### Known behavior

- OBI full-flow tests take ~15s each because `awaitListeners()` polls for listening ports with a 15s timeout. Our fake `sleep` processes never bind ports — this is expected.
- Node service name is empty in `ServiceSetting` because `OverrideServiceNameOnContainer=true` overwrites it with the container name (empty — no Docker socket inside the test container).
- Python service name falls through to `"python-service"` because Python's `extractServiceName` doesn't check systemd unit name.

## CI Workflows

| Workflow | File | Trigger | What it runs |
|----------|------|---------|-------------|
| Unit Tests | `.github/workflows/unit-test.yml` | PR to master | `go vet ./...` + `go test ./...` |
| Integration Tests | `.github/workflows/integration-test.yml` | PR to master | `tests/integration/run.sh` (Docker e2e) |
| PR Agent | `.github/workflows/pr-agent.yml` | PR events | Gemini AI code review |

## Adding New Tests

**New unit test**: Add `_test.go` file alongside the source in the same package. Use table-driven tests with `t.Run()`.

**New integration test**: Add to the existing `e2e_*_test.go` files in `pkg/otelinject/`. Must have `//go:build integration` tag. Use `t.Cleanup` for isolation.

**New fake service**: Add a `.service` file to `tests/integration/setup/`, copy the binary in the Dockerfile, enable it via `systemctl enable`, and add it to the service wait loop in `run.sh`.
