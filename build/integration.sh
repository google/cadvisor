#!/usr/bin/env bash

# Copyright 2016 Google Inc. All rights reserved.
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

log_file="cadvisor.log"
if [ "$#" -gt 0 ]; then
  log_file="$1"
fi

TEST_PID=$$
printf "" # Refresh sudo credentials if necessary.
function start {
  set +e  # We want to handle errors if cAdvisor crashes.
  echo ">> starting cAdvisor locally"
  GORACE="halt_on_error=1" ./cadvisor --docker_env_metadata_whitelist=TEST_VAR --v=4 --logtostderr &> "$log_file"
  if [ $? != 0 ]; then
    echo "!! cAdvisor exited unexpectedly with Exit $?"
    kill $TEST_PID # cAdvisor crashed: abort testing.
  fi
}
start &
RUNNER_PID=$!

function cleanup {
  if pgrep cadvisor > /dev/null; then
    echo ">> stopping cAdvisor"
    pkill -SIGINT cadvisor
    wait $RUNNER_PID
  fi
}
trap cleanup EXIT

readonly TIMEOUT=30 # Timeout to wait for cAdvisor, in seconds.
START=$(date +%s)
while [ "$(curl -Gs http://localhost:8080/healthz)" != "ok" ]; do
  if (( $(date +%s) - $START > $TIMEOUT )); then
    echo "Timed out waiting for cAdvisor to start"
    exit 1
  fi
  echo "Waiting for cAdvisor to start ..."
  sleep 1
done

echo ">> running integration tests against local cAdvisor"
./api.test --vmodule=*=2
./healthz.test --vmodule=*=2
