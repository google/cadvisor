#!/usr/bin/env bash

# Copyright 2016 Google Inc. All rights reserved.
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
# permissions if you cadvisor can't find containers.
# USE_SUDO=true make test-integration
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

# Diagnostic logging for docker/containerd debugging
echo ">> Diagnostic information:"
echo "=== Docker version ==="
docker version || echo "docker version failed"
echo "=== Docker info ==="
docker info || echo "docker info failed"
echo "=== Containerd socket check ==="
ls -la /run/containerd/ 2>/dev/null || echo "/run/containerd/ not found"
ls -la /var/run/containerd/ 2>/dev/null || echo "/var/run/containerd/ not found"
ls -la /var/run/docker/containerd/ 2>/dev/null || echo "/var/run/docker/containerd/ not found"
echo "=== Find all containerd sockets ==="
find /var/run /run -name "*.sock" 2>/dev/null | head -20 || echo "No sockets found"
echo "=== Docker socket check ==="
ls -la /var/run/docker.sock 2>/dev/null || echo "/var/run/docker.sock not found"
echo "=== Running processes (docker/containerd) ==="
ps aux | grep -E "(docker|containerd)" | grep -v grep || echo "No docker/containerd processes found"
echo "=== Kernel version ==="
uname -r
echo "=== End diagnostic information ==="

# Detect containerd socket path - Docker-in-Docker uses a different path
CONTAINERD_SOCK="/run/containerd/containerd.sock"
if [ -S "/run/docker/containerd/containerd.sock" ]; then
  CONTAINERD_SOCK="/run/docker/containerd/containerd.sock"
  echo ">> Using Docker-embedded containerd socket: $CONTAINERD_SOCK"
fi

function start {
  set +e  # We want to handle errors if cAdvisor crashes.
  echo ">> starting cAdvisor locally"
  cadvisor_prereqs=""
  if [ $USE_SUDO = true ]; then
    cadvisor_prereqs=sudo
  fi
  # cpu, cpuset, percpu, memory, disk, diskIO, network, perf_event metrics should be enabled.
  GORACE="halt_on_error=1" $cadvisor_prereqs $cadvisor_bin --enable_metrics="cpu,cpuset,percpu,memory,disk,diskIO,network,perf_event" --env_metadata_whitelist=TEST_VAR --containerd="$CONTAINERD_SOCK" --v=6 --logtostderr $CADVISOR_ARGS &> "$log_file"
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
    echo ">> configuring cgroupsv2 for docker in docker..."
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

echo ">> running integration tests against local cAdvisor"
if ! [ -f ./api.test ] || ! [ -f ./common.test ]; then
  echo You must compile the ./api.test binary and ./common.test binary before
  echo running the integration tests.
  exit 1
fi
./api.test --vmodule=*=2 -test.v
./common.test -test.v
