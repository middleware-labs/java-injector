FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

# Install systemd and required tools
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      systemd systemd-sysv dbus \
      ca-certificates curl && \
    # Strip unnecessary systemd units that slow container boot
    rm -f /etc/systemd/system/*.wants/* \
          /lib/systemd/system/local-fs.target.wants/* \
          /lib/systemd/system/sockets.target.wants/*udev* \
          /lib/systemd/system/sockets.target.wants/*initctl* \
          /lib/systemd/system/sysinit.target.wants/systemd-tmpfiles-setup* \
          /lib/systemd/system/systemd-update-utmp* && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Go 1.24.2
RUN curl -fsSL https://go.dev/dl/go1.24.2.linux-amd64.tar.gz | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:/go/bin:${PATH}"
ENV GOPATH="/go"

# Create fake language binaries (copies of sleep, not symlinks)
# readlink /proc/<pid>/exe will resolve to these paths, passing handler detection
RUN cp /bin/sleep /usr/local/bin/java && \
    cp /bin/sleep /usr/local/bin/node && \
    cp /bin/sleep /usr/local/bin/python3

# Stub agent assets for Java (ValidateJavaAgent checks these paths)
RUN mkdir -p /usr/lib/opentelemetry/jvm && \
    touch /usr/lib/opentelemetry/libotelinject.so && \
    touch /usr/lib/opentelemetry/jvm/javaagent.jar

# Stub agent assets for Node.js (ValidateNodeAgent checks these paths)
RUN mkdir -p /usr/lib/opentelemetry/nodejs/node_modules/@opentelemetry/auto-instrumentations-node/build/src && \
    mkdir -p /usr/lib/opentelemetry/nodejs/node_modules/require-in-the-middle && \
    mkdir -p /usr/lib/opentelemetry/nodejs/node_modules/import-in-the-middle && \
    mkdir -p /usr/lib/opentelemetry/nodejs/node_modules/@opentelemetry/sdk-node && \
    mkdir -p /usr/lib/opentelemetry/nodejs/node_modules/@opentelemetry/api && \
    touch /usr/lib/opentelemetry/nodejs/node_modules/@opentelemetry/auto-instrumentations-node/build/src/register.js && \
    echo '{"name":"@opentelemetry/auto-instrumentations-node","version":"0.99.0"}' > \
      /usr/lib/opentelemetry/nodejs/node_modules/@opentelemetry/auto-instrumentations-node/package.json

# Stub agent assets for Python (ValidatePythonAgent checks these paths)
# Ubuntu 22.04 uses glibc, so the flavor directory is "glibc"
RUN mkdir -p /opt/otel-python-agent/glibc/opentelemetry && \
    touch /opt/otel-python-agent/glibc/sitecustomize.py

# Fake OBI binary and config
RUN mkdir -p /etc/obi-agent && \
    cp /bin/sleep /usr/local/bin/obi

# Copy test service units
COPY tests/integration/setup/test-java.service /etc/systemd/system/test-java.service
COPY tests/integration/setup/test-node.service /etc/systemd/system/test-node.service
COPY tests/integration/setup/test-python.service /etc/systemd/system/test-python.service
COPY tests/integration/setup/obi-agent.service /etc/systemd/system/obi-agent.service

# Copy seed OBI config
COPY tests/integration/setup/seed-config.yaml /etc/obi-agent/config.yaml

# Enable test services so systemd starts them at boot
RUN systemctl enable test-java test-node test-python obi-agent

# Copy project source (go.mod first for layer caching)
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

VOLUME ["/sys/fs/cgroup"]
STOPSIGNAL SIGRTMIN+3
CMD ["/lib/systemd/systemd"]
