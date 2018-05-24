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

ifeq ($(ARCH),)
ARCH := amd64 #amd64 as the default architecture.
endif

ifeq ($(REV),)
REV := $(shell git rev-parse --short HEAD)
endif


GO := go
pkgs  = $(shell $(GO) list ./... | grep -v vendor)


all: presubmit build test

test:
	@echo ">> running tests"
	@$(GO) test -short -race $(pkgs)

test-integration:
	ARCH=$(ARCH) GO_FLAGS="-race" ./build/build.sh
	cp cadvisor.${ARCH} cadvisor
	go test -c github.com/google/cadvisor/integration/tests/api
	go test -c github.com/google/cadvisor/integration/tests/healthz
	ARCH=$(ARCH) ./build/integration.sh

test-runner:
	@$(GO) build github.com/google/cadvisor/integration/runner

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

build: assets
	@echo ">> building binaries"
	ARCH=$(ARCH) ./build/build.sh

assets:
	@echo ">> building assets"
	@./build/assets.sh

release:
	@echo ">> building release binaries"
	ARCH=$(ARCH) ./build/release.sh

docker: build
	@echo ">> building docker images"
	ARCH=$(ARCH) TAGS="-t cadvisor:$(REV)-$(ARCH)" ./build/docker.sh

presubmit: vet
	@echo ">> checking go formatting"
	@./build/check_gofmt.sh
	@echo ">> checking file boilerplate"
	@./build/check_boilerplate.sh

.PHONY: all build docker format release test test-integration vet presubmit
