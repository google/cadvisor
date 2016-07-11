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

RELEASE=${RELEASE:-false} # Whether to build for an official release.
GO_FLAGS=${GO_FLAGS:-}    # Extra go flags to use in the build.

repo_path="github.com/google/cadvisor"

version=$( git describe --tags --dirty --abbrev=14 | sed -E 's/-([0-9]+)-g/.\1+/' )
revision=$( git rev-parse --short HEAD 2> /dev/null || echo 'unknown' )
branch=$( git rev-parse --abbrev-ref HEAD 2> /dev/null || echo 'unknown' )
build_user="${USER}@${HOSTNAME}"
build_date=$( date +%Y%m%d-%H:%M:%S )
go_version=$( go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/' )

GO_CMD="install"

if [ "$RELEASE" == "true" ]; then
  # Only allow releases of tagged versions.
  TAGGED='^v[0-9]+\.[0-9]+\.[0-9]+$'
  if [[ ! "$version" =~ $TAGGED ]]; then
    echo "Only tagged versions are allowed for releases" >&2
    echo "Found: $version" >&2
    exit 1
  fi

  # Don't include hostname with release builds
  build_user="$(git config --get user.email)"
  build_date=$( date +%Y%m%d ) # Release date is only to day-granularity

  # Don't use cached build objects for releases.
  GO_CMD="build"
fi

# go 1.4 requires ldflags format to be "-X key value", not "-X key=value"
ldseparator="="
if [ "${go_version:0:3}" = "1.4" ]; then
	ldseparator=" "
fi

ldflags="
  -extldflags '-static'
  -X ${repo_path}/version.Version${ldseparator}${version}
  -X ${repo_path}/version.Revision${ldseparator}${revision}
  -X ${repo_path}/version.Branch${ldseparator}${branch}
  -X ${repo_path}/version.BuildUser${ldseparator}${build_user}
  -X ${repo_path}/version.BuildDate${ldseparator}${build_date}
  -X ${repo_path}/version.GoVersion${ldseparator}${go_version}"

echo ">> building cadvisor"

if [ "$RELEASE" == "true" ]; then
  echo "Building release candidate with -ldflags $ldflags"
fi

GOBIN=$PWD go "$GO_CMD" ${GO_FLAGS} -ldflags "${ldflags}" "${repo_path}"

exit 0
