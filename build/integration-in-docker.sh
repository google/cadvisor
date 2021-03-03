#!/usr/bin/env bash

# Copyright 2020 Google Inc. All rights reserved.
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
trap delete EXIT INT

function run_tests() {
  BUILD_CMD="env GOOS=linux GO_FLAGS='$GO_FLAGS' ./build/build.sh amd64 && \
    env GOOS=linux GOFLAGS='$GO_FLAGS' go test -c github.com/google/cadvisor/integration/tests/api && \
    env GOOS=linux GOFLAGS='$GO_FLAGS' go test -c github.com/google/cadvisor/integration/tests/healthz"

  if [ "$BUILD_PACKAGES" != "" ]; then
    BUILD_CMD="echo 'deb http://deb.debian.org/debian buster-backports main'>/etc/apt/sources.list.d/buster.list && \
    apt update && \
    apt install -y -t buster-backports $BUILD_PACKAGES && \
    $BUILD_CMD"
  fi
  docker run --rm \
    -w /go/src/github.com/google/cadvisor \
    -v ${PWD}:/go/src/github.com/google/cadvisor \
    golang:"$GOLANG_VERSION-buster" \
    bash -c "$BUILD_CMD"

  EXTRA_DOCKER_OPTS="-e DOCKER_IN_DOCKER_ENABLED=true"
  if [[ "${OSTYPE}" == "linux"* ]]; then
    EXTRA_DOCKER_OPTS+=" -v ${TMPDIR}/docker-graph:/docker-graph"
  fi

  mkdir ${TMPDIR}/docker-graph
  docker run --rm \
    -w /go/src/github.com/google/cadvisor \
    -v ${ROOT}:/go/src/github.com/google/cadvisor \
    ${EXTRA_DOCKER_OPTS} \
    --privileged \
    --cap-add="sys_admin" \
    --entrypoint="" \
    gcr.io/k8s-testimages/bootstrap \
    bash -c "echo 'deb http://deb.debian.org/debian buster-backports main'>/etc/apt/sources.list.d/buster.list && \
    cat /etc/apt/sources.list.d/buster.list && \
    apt update && \
    apt install -y -t buster-backports $PACKAGES && \
    CADVISOR_ARGS="$CADVISOR_ARGS" /usr/local/bin/runner.sh build/integration.sh"
}

GO_FLAGS=${GO_FLAGS:-"-tags=netgo -race"}
PACKAGES=${PACKAGES:-"sudo"}
BUILD_PACKAGES=${BUILD_PACKAGES:-}
CADVISOR_ARGS=${CADVISOR_ARGS:-}
GOLANG_VERSION=${GOLANG_VERSION:-"1.16"}
run_tests "$GO_FLAGS" "$PACKAGES" "$BUILD_PACKAGES" "$CADVISOR_ARGS"
