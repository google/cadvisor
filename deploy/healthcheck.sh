#!/bin/bash

# Copyright 2025 Google Inc. All rights reserved.
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

# Default port
PORT=8080

# Extract port from the cadvisor process command line
if [ -f /proc/1/cmdline ]; then
    CMDLINE=$(tr '\0' ' ' < /proc/1/cmdline)

    # Look for -port=XXXX or --port=XXXX
    for arg in $CMDLINE; do
        case "$arg" in
            -port=*)
                PORT="${arg#-port=}"
                ;;
            --port=*)
                PORT="${arg#--port=}"
                ;;
        esac
    done
fi

wget --quiet --tries=1 --spider "http://localhost:${PORT}/healthz" || exit 1
