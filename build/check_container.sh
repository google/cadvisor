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
#
# Description:
# This script is meant to run a basic test against each of the CPU architectures
# cadvisor should support.
#
# This script requires that you have run qemu-user-static so that your machine
# can interpret ELF binaries for other architectures using QEMU:
# https://github.com/multiarch/qemu-user-static#getting-started
#
# $ docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
#
# Usage:
# ./check_container.sh gcr.io/tstapler-gke-dev/cadvisor:v0.44.1-test-4
target_image=$1

# Architectures officially supported by cadvisor
arches=( "amd64" "arm" "arm64" "s390x" )

# Docker doesn't handle images with different architectures but the same tag.
# Remove the container and the image use by it to avoid problems.
cleanup() {
    echo Cleaning up the container $1
    docker stop $1
    docker rmi $target_image
    echo
}

for arch in "${arches[@]}"; do
  echo Testing that we can run $1 on $arch and curl the /healthz endpoint
  echo
  container_id=$(docker run --platform "linux/$arch" -p 8080:8080 --rm --detach "$target_image")
  docker_exit_code=$?
  if [ $docker_exit_code -ne 0 ]; then
    echo Failed to run container docker exited with $docker_exit_code
    cleanup $container_id
    exit $docker_exit_code
  fi
  sleep 10
  echo
  echo Testing the container with curl:
  curl --show-error --retry 5 --fail -L 127.0.0.1:8080/healthz
  echo
  echo
  curl_exit=$?
  if [ $curl_exit -ne 0 ]; then
    echo  Curling $target_image did not work
    cleanup $container_id
    exit $curl_exit
  fi
  echo Success!
  echo
  cleanup $container_id
done
