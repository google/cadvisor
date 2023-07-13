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

if ! CONTAINER_ENGINE=$(command -v docker || command -v podman); then
  echo "Neither docker nor podman found. Exiting."
  exit 1
fi

function run_tests() {
  BUILD_CMD="make test"
  if [ "$BUILD_PACKAGES" != "" ]; then
    BUILD_CMD="echo 'deb http://deb.debian.org/debian buster-backports main'>/etc/apt/sources.list.d/buster.list
    apt update
    apt install -y -t buster-backports $BUILD_PACKAGES
    $BUILD_CMD"
  fi

  $CONTAINER_ENGINE run --rm \
    -w /go/src/github.com/google/cadvisor \
    -v ${PWD}:/go/src/github.com/google/cadvisor \
    -e GO_FLAGS \
    golang:${GOLANG_VERSION} \
    bash -e -c "$BUILD_CMD"
}

GO_FLAGS=${GO_FLAGS:-"-tags=netgo -race"}
BUILD_PACKAGES=${BUILD_PACKAGES:-}
GOLANG_VERSION=${GOLANG_VERSION:-"1.20"}
run_tests
