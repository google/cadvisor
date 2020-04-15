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

export GO_FLAGS="-tags=libpfm,netgo -race"
export PACKAGES="sudo libpfm4"
export BUILD_PACKAGES="libpfm4 libpfm4-dev"
export CADVISOR_ARGS="-perf_events_config=perf/testing/perf-non-hardware.json"
