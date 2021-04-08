#!/bin/bash

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

# TODO(bobbypage): Replace this with `go mod tidy --check` when it exists:
# https://github.com/golang/go/issues/27005.

# Checks if go mod tidy changes are needed for a given go module.
# If changes are needed, prints the diff and exits 1 status code.
# Arguments:
#   Directory of go module
function lint_gotidy() {
    MODULE_DIRECTORY="$1"

    pushd "${MODULE_DIRECTORY}" > /dev/null
    TMP_GOMOD=$(mktemp)
    TMP_GOSUM=$(mktemp)

    # Make a copy of the current files
    cp go.mod "${TMP_GOMOD}"
    cp go.sum "${TMP_GOSUM}"

    go mod tidy

    DIFF_MOD=$(diff -u "${TMP_GOMOD}" go.mod)
    DIFF_SUM=$(diff -u "${TMP_GOSUM}" go.sum)

    # Copy the files back
    cp "${TMP_GOMOD}" go.mod
    cp "${TMP_GOSUM}" go.sum

    if [[ -n "${DIFF_MOD}" || -n "${DIFF_SUM}" ]]; then
        echo "go tidy changes are needed; please run make tidy"
        echo "go.mod diff:"
        echo "${DIFF_MOD}"
        echo "go.sum diff:"
        echo "${DIFF_SUM}"
        exit 1
    fi

    popd > /dev/null
}

# Check if go mod tidy changes needed on main module
lint_gotidy "."

# Check if go mod tidy changes needed on cmd module
lint_gotidy "cmd"
