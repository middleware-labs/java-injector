#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
IMAGE_NAME="mw-injector-integration-test"
CONTAINER_NAME="mw-injector-e2e"

echo "=== Building integration test image ==="
docker build -t "$IMAGE_NAME" -f "$SCRIPT_DIR/Dockerfile" "$PROJECT_ROOT"

echo "=== Removing any existing container ==="
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true

echo "=== Starting systemd container ==="
docker run -d \
  --name "$CONTAINER_NAME" \
  --privileged \
  --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  "$IMAGE_NAME"

echo "=== Waiting for systemd to boot ==="
for i in $(seq 1 30); do
  if docker exec "$CONTAINER_NAME" systemctl is-system-running --wait 2>/dev/null | grep -qE "running|degraded"; then
    echo "systemd is ready"
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "ERROR: systemd did not become ready in 30s"
    docker exec "$CONTAINER_NAME" systemctl status --no-pager 2>/dev/null || true
    docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
    exit 1
  fi
  sleep 1
done

echo "=== Waiting for test services ==="
for svc in test-java test-node test-python obi-agent; do
  for i in $(seq 1 15); do
    if docker exec "$CONTAINER_NAME" systemctl is-active --quiet "$svc" 2>/dev/null; then
      echo "  $svc is active"
      break
    fi
    if [ "$i" -eq 15 ]; then
      echo "ERROR: $svc did not become active in 15s"
      docker exec "$CONTAINER_NAME" systemctl status "$svc" --no-pager 2>/dev/null || true
      docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
      exit 1
    fi
    sleep 1
  done
done

echo "=== Running integration tests ==="
set +e
docker exec "$CONTAINER_NAME" \
  go test -v -tags integration -count=1 -timeout 5m ./pkg/otelinject/
exit_code=$?
set -e

echo "=== Cleaning up ==="
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true

exit $exit_code
