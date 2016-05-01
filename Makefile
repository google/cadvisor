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

GO := godep go
pkgs  = $(shell $(GO) list ./...)
SOURCEDIR = $(shell pwd)
SOURCES := $(shell find $(SOURCEDIR) -name '*.go')
DOCKER = $(shell command -v docker 2> /dev/null)
build-image = cadvisor-build-image
DOCKERMAKE = $(DOCKER) run --rm -i -v $(SOURCEDIR):/go/src/github.com/google/cadvisor $(build-image) make

all: format build test

all-docker: format build-docker test-docker

build-image: build/Dockerfile
	@$(DOCKER) build -t $(build-image) build/
	@touch $@

test-docker: $(SOURCES) build-image
	@echo ">> running tests using docker"
	@$(DOCKERMAKE) test

test: $(SOURCES)
	@echo ">> running tests"
	@$(GO) test -tags test -short -race $(pkgs)

test-integration: build test
	@./build/integration.sh

test-integration-docker: 
	@$(DOCKERMAKE) test-integration

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

build-docker: $(SOURCES) build-image 
	@echo ">> building binaries using docker"
	@$(DOCKERMAKE) build

build: $(SOURCES)
	@echo ">> building binaries"
	@./build/assets.sh && ./build/build.sh
	@touch $@

release: build
	@./build/release.sh

docker:
	@docker build -t cadvisor:$(shell git rev-parse --short HEAD) -f deploy/Dockerfile .

.PHONY: all format build test vet docker
