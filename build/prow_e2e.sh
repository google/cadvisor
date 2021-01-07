#!/usr/bin/env bash

# Copyright 2018 Google Inc. All rights reserved.
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
set -x

BUILDER=${BUILDER:-false} # Whether this is running a PR builder job.

export GO_FLAGS="-race"
export GORACE="halt_on_error=1"

# cd to cadvisor directory
parent_path=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )
cd "$parent_path/.."

# Check whether assets need to be rebuilt.
FORCE=true build/assets.sh
if [[ ! -z "$(git diff --name-only -- cmd/internal/pages)" ]]; then
  echo "Found changes to UI assets:"
  git diff --name-only -- cmd/internal/pages
  echo "Run: 'make assets FORCE=true'"
  exit 1
fi

make all

# compile integration tests so they can be run without go installed
go test -c github.com/google/cadvisor/integration/tests/api
go test -c github.com/google/cadvisor/integration/tests/healthz
