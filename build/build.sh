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

export GOOS=${GOOS:-$(go env GOOS)}
export GOARCH=${GOARCH:-$(go env GOARCH)}
export CGO_ENABLED=${GO_CGO_ENABLED:-"1"}
GO_FLAGS=${GO_FLAGS:-"-tags netgo"}    # Extra go flags to use in the build.
BUILD_USER=${BUILD_USER:-"${USER}@${HOSTNAME}"}
BUILD_DATE=${BUILD_DATE:-$( date +%Y%m%d-%H:%M:%S )}
VERBOSE=${VERBOSE:-}
OUTPUT_NAME_WITH_ARCH=${OUTPUT_NAME_WITH_ARCH:-"false"}

repo_path="github.com/google/cadvisor"

version=${VERSION:-$( git describe --tags --dirty --abbrev=14 | sed -E 's/-([0-9]+)-g/.\1+/' )}
revision=$( git rev-parse --short HEAD 2> /dev/null || echo 'unknown' )
branch=$( git rev-parse --abbrev-ref HEAD 2> /dev/null || echo 'unknown' )
go_version=$( go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/' )


# go 1.4 requires ldflags format to be "-X key value", not "-X key=value"
ldseparator="="
if [ "${go_version:0:3}" = "1.4" ]; then
	ldseparator=" "
fi

ldflags="
  -X ${repo_path}/version.Version${ldseparator}${version}
  -X ${repo_path}/version.Revision${ldseparator}${revision}
  -X ${repo_path}/version.Branch${ldseparator}${branch}
  -X ${repo_path}/version.BuildUser${ldseparator}${BUILD_USER}
  -X ${repo_path}/version.BuildDate${ldseparator}${BUILD_DATE}
  -X ${repo_path}/version.GoVersion${ldseparator}${go_version}"

echo ">> building cadvisor"

if [ -n "$VERBOSE" ]; then
  echo "Building with -ldflags $ldflags"
fi

mkdir -p "$PWD/_output"
output_file="$PWD/_output/cadvisor"
if [ "${OUTPUT_NAME_WITH_ARCH}" = "true" ] ; then
  output_file="${output_file}-${version}-${GOOS}-${GOARCH}"
fi

# Since github.com/google/cadvisor/cmd is a submodule, we must build from inside that directory
pushd cmd > /dev/null
  go build ${GO_FLAGS} -ldflags "${ldflags}" -o "${output_file}" "${repo_path}/cmd"
popd > /dev/null

exit 0
