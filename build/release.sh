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

if [ -z "$VERSION" ]; then
  VERSION=$( git describe --tags --dirty --abbrev=14 | sed -E 's/-([0-9]+)-g/.\1+/' )
  # Only allow releases of tagged versions.
  TAGGED='^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)\.?[0-9]*)?$'
  if [[ ! "$VERSION" =~ $TAGGED ]]; then
    echo "Error: Only tagged versions are allowed for releases" >&2
    echo "Found: $VERSION" >&2
    exit 1
  fi
fi

read -p "Please confirm: $VERSION is the desired version (Type y/n to continue):" -n 1 -r
echo
if ! [[ $REPLY =~ ^[Yy]$ ]]; then
  exit 1
fi

# Don't include hostname with release builds
if ! git_user="$(git config --get user.email)"; then
  echo "Error: git user not set, use:"
  echo "git config user.email <email>"
  exit 1
fi

export BUILD_USER="$git_user"
export BUILD_DATE=$( date +%Y%m%d ) # Release date is only to day-granularity
export VERBOSE=true

# Build the docker image
echo ">> building cadvisor docker image"
image_name=${IMAGE_NAME:-"gcr.io/cadvisor/cadvisor"}
final_image="$image_name:${VERSION}"

docker buildx inspect cadvisor-builder > /dev/null \
|| docker buildx create --name cadvisor-builder --use

# Build binaries

# A mapping of the docker arch name to the qemu arch name
declare -A arches=( ["amd64"]="x86_64" ["arm"]="arm" ["arm64"]="aarch64" ["s390x"]="s390x")

for arch in "${arches[@]}"; do
  if ! hash "qemu-${arch}-static"; then
    echo Releasing multi arch containers requires qemu-user-static.
    echo
    echo Please install using apt-get install qemu-user-static or
    echo a similar package for your OS.

    exit 1
  fi
done

for arch in "${!arches[@]}"; do
  GOARCH="$arch" GO_CGO_ENABLED="0" OUTPUT_NAME_WITH_ARCH="true" build/build.sh
  arch_specific_image="${image_name}-${arch}:${VERSION}"
  docker buildx build --platform "linux/${arch}" --build-arg VERSION="$VERSION" -f deploy/Dockerfile -t "$arch_specific_image"  --progress plain --push .
  docker manifest create --amend "$final_image" "$arch_specific_image"
  docker manifest annotate --os=linux --arch="$arch" "$final_image" "$arch_specific_image"
done
docker manifest push "$final_image"
echo
echo "Release info (copy to the release page)":
echo
echo Multi Arch Container Image:
echo $final_image
echo
echo Architecture Specific Container Images:
for arch in "${!arches[@]}"; do
  echo "${image_name}-${arch}:${VERSION}"
done
echo
echo Binaries:
(cd _output && find . -name "cadvisor-${VERSION}*" -exec sha256sum --tag {} \;)
exit 0
