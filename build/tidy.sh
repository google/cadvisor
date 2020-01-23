#!/usr/bin/env bash

# Copyright 2015 Google Inc. All rights reserved.
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

# info has no dependencies
pushd info >/dev/null
   go mod tidy
popd >/dev/null

# registry depends only on info
pushd registry >/dev/null
  go mod tidy
popd >/dev/null

# storage depends only on info,registry
pushd storage >/dev/null
  go mod tidy
popd >/dev/null

# root depends only on info,registry
go mod tidy

# cmd depends on everything
pushd cmd >/dev/null
  go mod tidy
popd >/dev/null
