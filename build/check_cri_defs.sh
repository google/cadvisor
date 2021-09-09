#!/usr/bin/env bash

# Copyright 2021 Google Inc. All rights reserved.
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

# Presubmit script that ensures that https://github.com/kubernetes/cri-api
# definitions that are copied into cAdvisor codebase exactly match the ones upstream.

set -e

function check_git_dirty() {
  if ! [ -z "$(git status --porcelain)" ]; then
    echo ">>> working tree is not clean"
    echo ">>> git status:"
    echo "$(git status)"
    exit 1
  fi
}

GIT_ROOT=$(dirname "${BASH_SOURCE}")/..

check_git_dirty

echo ">>> updating k8s CRI definitions..."
"${GIT_ROOT}/build/update_cri_defs.sh"

# If k8s CRI definitions were manually modified or don't match the ones
# upstream, git tree will be unclean.
echo ">>> checking git tree clean after updating k8s CRI definitions..."
check_git_dirty
