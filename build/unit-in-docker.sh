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

#set -ex
set -x
function run_tests() {
  BUILD_CMD="go test $GO_FLAGS $(go list $GO_FLAGS ./... | grep -v 'vendor\|integration' | tr '\n' ' ') && \
    cd cmd && go test $GO_FLAGS $(go list $GO_FLAGS ./... | grep -v 'vendor\|integration' | tr '\n' ' ')"
  if [ "$BUILD_PACKAGES" != "" ]; then
    BUILD_CMD="apt-get update && apt-get install $BUILD_PACKAGES && \
    $BUILD_CMD"
  fi

  docker run --rm \
    -w /go/src/github.com/google/cadvisor \
    -v ${PWD}:/go/src/github.com/google/cadvisor \
    golang:${GOLANG_VERSION} \
    bash -c "$BUILD_CMD"
}

GO_FLAGS=${GO_FLAGS:-"-tags=netgo -race"}
BUILD_PACKAGES=${BUILD_PACKAGES:-}
GOLANG_VERSION=${GOLANG_VERSION:-"1.14"}
run_tests
