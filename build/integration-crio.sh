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

set -e
# When running this script locally, you may need to run cadvisor with sudo
# permissions if cadvisor can't find containers.
# USE_SUDO=true make test-integration-crio
USE_SUDO=${USE_SUDO:-false}
cadvisor_bin=${CADVISOR_BIN:-"./_output/cadvisor"}

if ! [ -f "$cadvisor_bin" ]; then
  echo Failed to find cadvisor binary for integration test at path $cadvisor_bin
  exit 1
fi

log_file="cadvisor.log"
if [ "$#" -gt 0 ]; then
  log_file="$1"
fi

TEST_PID=$$
printf "" # Refresh sudo credentials if necessary.

# Check if CRI-O is available and start it if not running
CRIO_SOCK="/var/run/crio/crio.sock"

# Function to start CRI-O if not running (for CI environments)
start_crio_if_needed() {
  if [ -S "$CRIO_SOCK" ]; then
    echo ">> CRI-O already running"
    return 0
  fi

  echo ">> CRI-O not running, attempting to start..."

  # Check if crio binary exists
  CRIO_BIN=""
  if command -v crio >/dev/null 2>&1; then
    CRIO_BIN="crio"
  elif [ -x /usr/local/bin/crio ]; then
    CRIO_BIN="/usr/local/bin/crio"
  fi

  if [ -z "$CRIO_BIN" ]; then
    echo "!! crio binary not found"
    return 1
  fi

  # Create required directories
  mkdir -p /var/run/crio /var/lib/containers/storage /var/log/crio/pods /run/containers/storage /etc/crio

  # Install conmon if not available (CRI-O requires it)
  if ! command -v conmon >/dev/null 2>&1 && [ ! -x /usr/local/bin/conmon ]; then
    echo ">> Installing conmon..."
    CONMON_VERSION=v2.1.8
    curl -L https://github.com/containers/conmon/releases/download/${CONMON_VERSION}/conmon.amd64 -o /usr/local/bin/conmon 2>/dev/null && \
    chmod +x /usr/local/bin/conmon
  fi

  # Ensure conmon is in PATH
  if [ -x /usr/local/bin/conmon ]; then
    export PATH=/usr/local/bin:$PATH
  fi

  # Install CNI plugins if not available
  if [ ! -d /opt/cni/bin ] || [ -z "$(ls -A /opt/cni/bin 2>/dev/null)" ]; then
    echo ">> Installing CNI plugins..."
    CNI_VERSION=v1.3.0
    mkdir -p /opt/cni/bin
    curl -L "https://github.com/containernetworking/plugins/releases/download/${CNI_VERSION}/cni-plugins-linux-amd64-${CNI_VERSION}.tgz" | tar -xz -C /opt/cni/bin
  fi

  # Create CNI config for bridge networking
  mkdir -p /etc/cni/net.d
  if [ ! -f /etc/cni/net.d/10-bridge.conf ]; then
    echo '{"cniVersion":"0.4.0","name":"bridge","type":"bridge","bridge":"cni0","isGateway":true,"ipMasq":true,"ipam":{"type":"host-local","subnet":"10.88.0.0/16","routes":[{"dst":"0.0.0.0/0"}]}}' > /etc/cni/net.d/10-bridge.conf
    echo '{"cniVersion":"0.4.0","name":"loopback","type":"loopback"}' > /etc/cni/net.d/99-loopback.conf
  fi

  # Create CRI-O config (always recreate to ensure correct settings)
  echo ">> Creating CRI-O config..."
  # Use vfs storage driver for Docker-in-Docker (overlay on overlay doesn't work)
  cat > /etc/crio/crio.conf << 'EOF'
[crio]
root = "/var/lib/containers/storage"
runroot = "/var/run/containers/storage"
log_dir = "/var/log/crio/pods"
version_file = "/var/run/crio/version"
storage_driver = "vfs"

[crio.api]
listen = "/var/run/crio/crio.sock"

[crio.runtime]
default_runtime = "crun"
[crio.runtime.runtimes.crun]
runtime_path = "/usr/bin/crun"
runtime_type = "oci"
runtime_root = "/run/crun"

[crio.image]
pause_image = "registry.k8s.io/pause:3.9"
EOF

  # Also configure containers storage
  mkdir -p /etc/containers
  cat > /etc/containers/storage.conf << 'EOF'
[storage]
driver = "vfs"
runroot = "/var/run/containers/storage"
graphroot = "/var/lib/containers/storage"
EOF

  # Create containers policy (allow all images) if it doesn't exist
  if [ ! -f /etc/containers/policy.json ]; then
    echo '{"default":[{"type":"insecureAcceptAnything"}]}' > /etc/containers/policy.json
  fi

  # Verify policy.json exists and is valid
  echo ">> Verifying /etc/containers/policy.json..."
  cat /etc/containers/policy.json

  # Start CRI-O
  echo ">> Starting CRI-O daemon..."
  $CRIO_BIN --log-level debug &
  CRIO_PID=$!
  sleep 2

  # Wait for CRI-O to be ready
  echo ">> Waiting for CRI-O to start..."
  CRICTL_BIN="crictl"
  if [ -x /usr/local/bin/crictl ]; then
    CRICTL_BIN="/usr/local/bin/crictl"
  fi

  for i in $(seq 1 60); do
    if $CRICTL_BIN info >/dev/null 2>&1; then
      echo ">> CRI-O is ready"
      return 0
    fi
    echo "Waiting for CRI-O... attempt $i"
    sleep 1
  done

  echo "!! CRI-O failed to start"
  return 1
}

# Try to start CRI-O if in Docker-in-Docker environment
if [[ "${DOCKER_IN_DOCKER_ENABLED:-}" == "true" ]]; then
  start_crio_if_needed
fi

# Diagnostic logging for CRI-O debugging
echo ">> Diagnostic information:"
echo "=== CRI-O version ==="
crio --version 2>/dev/null || /usr/local/bin/crio --version 2>/dev/null || echo "crio --version failed"
echo "=== crictl version ==="
crictl version 2>/dev/null || /usr/local/bin/crictl version 2>/dev/null || echo "crictl version failed"
echo "=== CRI-O socket check ==="
ls -la /var/run/crio/ 2>/dev/null || echo "/var/run/crio/ not found"
ls -la /run/crio/ 2>/dev/null || echo "/run/crio/ not found"
echo "=== CRI-O info ==="
crictl info 2>/dev/null || /usr/local/bin/crictl info 2>/dev/null || echo "crictl info failed"
echo "=== Running processes (crio) ==="
ps aux | grep -E "crio" | grep -v grep || echo "No crio processes found"
echo "=== Kernel version ==="
uname -r
echo "=== End diagnostic information ==="

# CRI-O socket is hardcoded in cAdvisor to /var/run/crio/crio.sock
if [ -S "$CRIO_SOCK" ]; then
  echo ">> CRI-O socket found at: $CRIO_SOCK"
else
  echo "!! CRI-O socket not found at: $CRIO_SOCK"
  echo "!! CRI-O may not be running"
fi

function start {
  set +e  # We want to handle errors if cAdvisor crashes.
  echo ">> starting cAdvisor locally"
  cadvisor_prereqs=""
  if [ $USE_SUDO = true ]; then
    cadvisor_prereqs=sudo
  fi
  # cpu, cpuset, percpu, memory, disk, diskIO, network metrics should be enabled.
  GORACE="halt_on_error=1" $cadvisor_prereqs $cadvisor_bin --enable_metrics="cpu,cpuset,percpu,memory,disk,diskIO,network" --env_metadata_whitelist=TEST_VAR --v=6 --logtostderr $CADVISOR_ARGS &> "$log_file"
  exit_code=$?
  if [ $exit_code != 0 ]; then
    echo "!! cAdvisor exited unexpectedly with Exit $exit_code"
    cat $log_file
    kill $TEST_PID # cAdvisor crashed: abort testing.
  fi
}
start &
RUNNER_PID=$!

function cleanup {
  if pgrep cadvisor > /dev/null; then
    echo ">> stopping cAdvisor"
    pkill -SIGINT cadvisor
    wait $RUNNER_PID
  fi
}
trap cleanup EXIT SIGINT TERM

readonly TIMEOUT=30 # Timeout to wait for cAdvisor, in seconds.
START=$(date +%s)
while [ "$(curl -Gs http://localhost:8080/healthz)" != "ok" ]; do
  if (( $(date +%s) - $START > $TIMEOUT )); then
    echo "Timed out waiting for cAdvisor to start"
    exit 1
  fi
  echo "Waiting for cAdvisor to start ..."
  sleep 1
done

if [[ "${DOCKER_IN_DOCKER_ENABLED:-}" == "true" ]]; then
  # see https://github.com/moby/moby/blob/master/hack/dind
  # cgroup v2: enable nesting
  if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
    echo ">> configuring cgroupsv2 for crio in container..."
    # move the processes from the root group to the /init group,
    # otherwise writing subtree_control fails with EBUSY.
    # An error during moving non-existent process (i.e., "cat") is ignored.
    mkdir -p /sys/fs/cgroup/init
    xargs -rn1 < /sys/fs/cgroup/cgroup.procs > /sys/fs/cgroup/init/cgroup.procs || :
    # enable controllers
    sed -e 's/ / +/g' -e 's/^/+/' < /sys/fs/cgroup/cgroup.controllers \
      > /sys/fs/cgroup/cgroup.subtree_control
  fi
fi

echo ">> running CRI-O integration tests against local cAdvisor"
if ! [ -f ./crio.test ]; then
  echo You must compile the ./crio.test binary before
  echo running the integration tests.
  exit 1
fi
./crio.test --vmodule=*=2 -test.v

echo ">> running common integration tests against local cAdvisor"
if [ -f ./common.test ]; then
  ./common.test -test.v
else
  echo "Skipping common tests (./common.test not found)"
fi
