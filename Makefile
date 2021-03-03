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

GO := go
pkgs     = $(shell $(GO) list ./... | grep -v vendor)
cmd_pkgs = $(shell cd cmd && $(GO) list ./... | grep -v vendor)
arch ?= $(shell go env GOARCH)
go_path = $(shell go env GOPATH)

ifeq ($(arch), amd64)
  Dockerfile_tag := ''
else
  Dockerfile_tag := '.''$(arch)'
endif


all: presubmit build test

test:
	@echo ">> running tests"
	@$(GO) test -short -race $(pkgs)
	@cd cmd && $(GO) test -short -race $(cmd_pkgs)

test-with-libpfm:
	@echo ">> running tests"
	@$(GO) test -short -race -tags="libpfm" $(pkgs)
	@cd cmd && $(GO) test -short -race -tags="libpfm" $(cmd_pkgs)

container-test:
	@echo ">> runinng tests in a container"
	@./build/unit-in-container.sh

docker-test: container-test
	@echo "docker-test target is deprecated, use container-test instead"

test-integration:
	@GO_FLAGS=${$GO_FLAGS:-"-race"} ./build/build.sh
	go test -c github.com/google/cadvisor/integration/tests/api
	go test -c github.com/google/cadvisor/integration/tests/healthz
	@./build/integration.sh

docker-test-integration:
	@./build/integration-in-docker.sh

test-runner:
	@$(GO) build github.com/google/cadvisor/integration/runner

tidy:
	@$(GO) mod tidy
	@cd cmd && $(GO) mod tidy

format:
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)
	@cd cmd && $(GO) fmt $(cmd_pkgs)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)
	@cd cmd && $(GO) vet $(cmd_pkgs)

build: assets
	@echo ">> building binaries"
	@./build/build.sh $(arch)

assets:
	@echo ">> building assets"
	@./build/assets.sh

release:
	@echo ">> building release binaries"
	@./build/release.sh

docker-%:
	@docker build -t cadvisor:$(shell git rev-parse --short HEAD) -f deploy/Dockerfile$(Dockerfile_tag) .

docker-build:
	@docker run --rm -w /go/src/github.com/google/cadvisor -v ${PWD}:/go/src/github.com/google/cadvisor golang:1.16 make build

presubmit: vet
	@echo ">> checking go formatting"
	@./build/check_gofmt.sh
	@echo ">> checking go mod tidy"
	@./build/check_gotidy.sh
	@echo ">> checking file boilerplate"
	@./build/check_boilerplate.sh

lint:
	@echo ">> running golangci-lint using configuration at .golangci.yml"
	@GOFLAGS="$(GO_FLAGS)" $(go_path)/bin/golangci-lint run

clean:
	@rm -f *.test cadvisor

.PHONY: all build docker format release test test-integration vet presubmit tidy
