#!/bin/sh

# Copyright 2026 Google Inc. All rights reserved.
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

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

TMPDIR=$(mktemp -d)
trap 'rm -rf "${TMPDIR}"' EXIT

run_case() {
  name="$1"
  expected_port="$2"
  shift 2

  cmdline_file="${TMPDIR}/${name}.cmdline"
  wget_log="${TMPDIR}/${name}.wget"

  : > "${cmdline_file}"
  for arg in "$@"; do
    printf '%s\0' "${arg}" >> "${cmdline_file}"
  done

  cat > "${TMPDIR}/wget" <<EOF
#!/bin/sh
printf '%s\n' "\$*" > "${wget_log}"
case "\$*" in
  *"http://localhost:${expected_port}/healthz"*) exit 0 ;;
  *) exit 1 ;;
esac
EOF
  chmod +x "${TMPDIR}/wget"

  PATH="${TMPDIR}:${PATH}" \
    CADVISOR_HEALTHCHECK_CMDLINE_FILE="${cmdline_file}" \
    sh "${SCRIPT_DIR}/healthcheck.sh"
}

run_case default 8080 cadvisor
run_case single_dash_equals 9090 cadvisor -port=9090
run_case double_dash_equals 9091 cadvisor --port=9091
run_case single_dash_space 9092 cadvisor -port 9092
run_case double_dash_space 9093 cadvisor --port 9093
