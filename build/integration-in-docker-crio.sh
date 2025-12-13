#!/usr/bin/env bash

# Copyright 2024 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

ROOT="$(cd "$(dirname "${BASH_SOURCE}")/.." && pwd -P)"
TMPDIR=$(mktemp -d)
function delete() {
  echo "Deleting ${TMPDIR}..."
  if [[ $EUID -ne 0 ]]; then
    sudo rm -rf "${TMPDIR}"
  else
    rm -rf "${TMPDIR}"
  fi
}
trap delete EXIT INT TERM

function run_tests() {

  # Detect architecture - the bootstrap image is amd64-only
  DOCKER_PLATFORM="linux/amd64"

  # Add safe.directory as workaround for https://github.com/actions/runner/issues/2033
  # Build for amd64 to match the test container
  BUILD_CMD="git config --global safe.directory /go/src/github.com/google/cadvisor && env GOOS=linux GOARCH=amd64 GO_FLAGS='$GO_FLAGS' CGO_ENABLED=0 ./build/build.sh && \
    env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -c github.com/google/cadvisor/integration/tests/crio"

  if [ "$BUILD_PACKAGES" != "" ]; then
    BUILD_CMD="apt update && apt install -y $BUILD_PACKAGES && \
    $BUILD_CMD"
  fi

  # Build in amd64 container to match test environment
  docker run --rm \
    --platform ${DOCKER_PLATFORM} \
    -w /go/src/github.com/google/cadvisor \
    -v ${PWD}:/go/src/github.com/google/cadvisor \
    golang:"$GOLANG_VERSION-bookworm" \
    bash -c "$BUILD_CMD"

  EXTRA_DOCKER_OPTS="-e DOCKER_IN_DOCKER_ENABLED=true"
  if [[ "${OSTYPE}" == "linux"* ]]; then
    EXTRA_DOCKER_OPTS+=" -v ${TMPDIR}/crio-graph:/var/lib/containers"
  fi

  mkdir -p ${TMPDIR}/crio-graph

  # Run tests in a privileged container with CRI-O
  # Use --platform to ensure consistent architecture
  # Use --cgroupns=host and --pid=host to share host namespaces (required for systemd cgroup manager)
  # Mount host's systemd and dbus sockets so CRI-O can use systemd cgroup manager
  docker run --rm \
    --platform ${DOCKER_PLATFORM} \
    -w /go/src/github.com/google/cadvisor \
    -v ${ROOT}:/go/src/github.com/google/cadvisor \
    -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
    -v /run/systemd:/run/systemd:ro \
    -v /run/dbus:/run/dbus:ro \
    ${EXTRA_DOCKER_OPTS} \
    --privileged \
    --cap-add="sys_admin" \
    --cgroupns=host \
    --pid=host \
    --entrypoint="" \
    gcr.io/k8s-staging-test-infra/bootstrap:v20250702-52f5173c3a \
    bash -c "export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get install -y $PACKAGES curl conntrack iptables dbus && \

    # Check if host systemd and dbus are available
    echo 'Checking host systemd and dbus...' && \
    ls -la /run/systemd/ || true && \
    ls -la /run/dbus/ || true && \
    systemctl --version || true && \

    # Install CRI-O and crictl from static binaries (more reliable than apt)
    CRIO_VERSION=v1.28.0 && \
    CRICTL_VERSION=v1.28.0 && \

    # Download and install crictl
    echo 'Installing crictl...' && \
    curl -L https://github.com/kubernetes-sigs/cri-tools/releases/download/\${CRICTL_VERSION}/crictl-\${CRICTL_VERSION}-linux-amd64.tar.gz | tar -C /usr/local/bin -xz && \
    chmod +x /usr/local/bin/crictl && \

    # Download and install CRI-O from Google Cloud Storage
    echo 'Installing CRI-O...' && \
    mkdir -p /tmp/crio-install && \
    curl -L https://storage.googleapis.com/cri-o/artifacts/cri-o.amd64.\${CRIO_VERSION}.tar.gz | tar -xz -C /tmp/crio-install && \
    mkdir -p /usr/local/bin /etc/crio /etc/containers && \
    ls -la /tmp/crio-install/cri-o/bin/ && \
    cp /tmp/crio-install/cri-o/bin/crio /usr/local/bin/ && \
    cp /tmp/crio-install/cri-o/bin/crio-status /usr/local/bin/ 2>/dev/null || true && \
    cp /tmp/crio-install/cri-o/bin/pinns /usr/local/bin/ 2>/dev/null || true && \
    cp /tmp/crio-install/cri-o/bin/conmon /usr/local/bin/ 2>/dev/null || true && \
    cp /tmp/crio-install/cri-o/bin/conmonrs /usr/local/bin/ 2>/dev/null || true && \
    chmod +x /usr/local/bin/crio* /usr/local/bin/pinns /usr/local/bin/conmon* 2>/dev/null || true && \
    ls -la /usr/local/bin/crio* /usr/local/bin/conmon* /usr/local/bin/pinns 2>/dev/null && \

    # Create CRI-O config with systemd cgroup manager (using host systemd via mounted socket)
    mkdir -p /etc/crio/crio.conf.d && \
    cat > /etc/crio/crio.conf <<CRIOEOF
[crio]
root = \"/var/lib/containers/storage\"
runroot = \"/var/run/containers/storage\"
log_dir = \"/var/log/crio/pods\"
version_file = \"/var/run/crio/version\"

[crio.api]
listen = \"/var/run/crio/crio.sock\"

[crio.runtime]
cgroup_manager = \"systemd\"
conmon_cgroup = \"pod\"
default_runtime = \"crun\"
[crio.runtime.runtimes.crun]
runtime_path = \"/usr/bin/crun\"
runtime_type = \"oci\"
runtime_root = \"/run/crun\"

[crio.image]
pause_image = \"registry.k8s.io/pause:3.9\"
CRIOEOF

    # Install crun as the runtime
    apt-get install -y crun && \

    # Create containers policy (required for image pulling)
    mkdir -p /etc/containers && \
    echo '{\"default\":[{\"type\":\"insecureAcceptAnything\"}]}' > /etc/containers/policy.json && \
    echo 'Created /etc/containers/policy.json:' && \
    cat /etc/containers/policy.json && \

    # Install CNI plugins
    echo 'Installing CNI plugins...' && \
    CNI_VERSION=v1.3.0 && \
    mkdir -p /opt/cni/bin && \
    curl -L \"https://github.com/containernetworking/plugins/releases/download/\${CNI_VERSION}/cni-plugins-linux-amd64-\${CNI_VERSION}.tgz\" | tar -xz -C /opt/cni/bin && \
    ls -la /opt/cni/bin/ && \

    # Create CNI config for bridge networking
    mkdir -p /etc/cni/net.d && \
    echo '{\"cniVersion\":\"0.4.0\",\"name\":\"bridge\",\"type\":\"bridge\",\"bridge\":\"cni0\",\"isGateway\":true,\"ipMasq\":true,\"ipam\":{\"type\":\"host-local\",\"subnet\":\"10.88.0.0/16\",\"routes\":[{\"dst\":\"0.0.0.0/0\"}]}}' > /etc/cni/net.d/10-bridge.conf && \
    echo '{\"cniVersion\":\"0.4.0\",\"name\":\"loopback\",\"type\":\"loopback\"}' > /etc/cni/net.d/99-loopback.conf && \
    echo 'CNI config created:' && \
    cat /etc/cni/net.d/10-bridge.conf && \

    # Configure crictl to use CRI-O socket
    cat > /etc/crictl.yaml <<EOF
runtime-endpoint: unix:///var/run/crio/crio.sock
image-endpoint: unix:///var/run/crio/crio.sock
timeout: 30
debug: true
EOF

    # Start CRI-O daemon in background
    mkdir -p /var/run/crio /var/lib/containers/storage /var/log/crio/pods /run/containers/storage && \
    echo 'Starting CRI-O...' && \
    /usr/local/bin/crio --log-level debug &
    CRIO_PID=\$! && \
    sleep 2 && \

    # Wait for CRI-O to be ready
    echo 'Waiting for CRI-O to start...' && \
    for i in \$(seq 1 60); do \
      if /usr/local/bin/crictl info >/dev/null 2>&1; then \
        echo 'CRI-O is ready'; \
        break; \
      fi; \
      echo \"Waiting for CRI-O... attempt \$i\"; \
      sleep 1; \
    done && \

    # Verify CRI-O is running
    /usr/local/bin/crictl info && \

    # Pull required images
    echo 'Pulling test images...' && \
    /usr/local/bin/crictl pull registry.k8s.io/pause:3.9 || true && \
    /usr/local/bin/crictl pull registry.k8s.io/busybox:1.27 || true && \

    # Add /usr/local/bin to PATH for the test runner
    export PATH=/usr/local/bin:\$PATH && \

    # Run the integration tests
    CADVISOR_ARGS='$CADVISOR_ARGS' /usr/local/bin/runner.sh build/integration-crio.sh"
}

# Note: -race requires CGO, but cross-compilation with CGO is problematic
# So we use -tags=netgo without -race for cross-platform compatibility
GO_FLAGS=${GO_FLAGS:-"-tags=netgo"}
PACKAGES=${PACKAGES:-"sudo"}
BUILD_PACKAGES=${BUILD_PACKAGES:-}
CADVISOR_ARGS=${CADVISOR_ARGS:-}
GOLANG_VERSION=${GOLANG_VERSION:-"1.25"}
run_tests
