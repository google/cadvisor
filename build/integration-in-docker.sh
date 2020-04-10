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

set -e

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

docker run --rm \
  -w /go/src/github.com/google/cadvisor \
  -v ${PWD}:/go/src/github.com/google/cadvisor \
  golang:1.13 \
  bash -c "env GOOS=linux GO_FLAGS='-race' ./build/build.sh amd64 && \
    env GOOS=linux go test -c github.com/google/cadvisor/integration/tests/api &&  \
    env GOOS=linux go test -c github.com/google/cadvisor/integration/tests/healthz"

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
  --entrypoint="" \
  gcr.io/k8s-testimages/bootstrap \
  bash -c "apt update && apt install sudo && \
/usr/local/bin/runner.sh build/integration.sh"
