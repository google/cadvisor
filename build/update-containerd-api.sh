#!/usr/bin/env bash

# Copyright 2022 Google Inc. All rights reserved.
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

set -o errexit
set -o nounset
set -o pipefail

GIT_ROOT=$(dirname "${BASH_SOURCE}")/..
CONTAINERD_VERSION=v1.6.6
CONTAINERD_TAR_GZ="https://github.com/containerd/containerd/archive/refs/tags/${CONTAINERD_VERSION}.tar.gz"

rm -rf "$GIT_ROOT/third_party/containerd/api"
mkdir -p "$GIT_ROOT/third_party/containerd"

pushd "$GIT_ROOT/third_party/containerd"
curl -sSL "$CONTAINERD_TAR_GZ" | tar --wildcards --strip-components=1 --exclude="vendor/*" -xzf - "*/api/" "*/LICENSE" "*/NOTICE"
popd

find "$GIT_ROOT/third_party/containerd/api" -name "*.go" \
  -exec sed -i "s|tasktypes \"github.com/containerd/containerd/api/types/task\"|tasktypes \"github.com/google/cadvisor/third_party/containerd/api/types/task\"|" {} \;

find "$GIT_ROOT/third_party/containerd/api" -name "*.go" \
  -exec sed -i "s|task \"github.com/containerd/containerd/api/types/task\"|task \"github.com/google/cadvisor/third_party/containerd/api/types/task\"|" {} \;

find "$GIT_ROOT/third_party/containerd/api" -name "*.go" \
  -exec sed -i "s|types \"github.com/containerd/containerd/api/types\"|types \"github.com/google/cadvisor/third_party/containerd/api/types\"|" {} \;

go mod tidy
pushd "$GIT_ROOT/cmd"
go mod tidy
popd
