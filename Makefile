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
GOLANGCI_VER := v1.51.2
GO_TEST ?= $(GO) test $(or $(GO_FLAGS),-race)
arch ?= $(shell go env GOARCH)

all: presubmit build test

test:
	@echo ">> running tests"
	@# Filter out integration.
	$(GO) list ./... | grep -vw integration | xargs $(GO_TEST)
	cd cmd && $(GO_TEST) ./...

test-with-libpfm: GO_FLAGS=-race -tags libpfm
test-with-libpfm: test

container-test:
	@echo ">> runinng tests in a container"
	@./build/unit-in-container.sh

docker-test: container-test
	@echo "docker-test target is deprecated, use container-test instead"

test-integration:
	GO_FLAGS=$(or $(GO_FLAGS),-race) ./build/build.sh
	$(GO_TEST) -c github.com/google/cadvisor/integration/tests/api
	$(GO_TEST) -c github.com/google/cadvisor/integration/tests/healthz
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
	@# goimports is a superset of gofmt.
	@goimports -w -local github.com/google/cadvisor .

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
	@docker build -t cadvisor:$(shell git rev-parse --short HEAD) -f deploy/Dockerfile .

docker-build:
	@docker run --rm -w /go/src/github.com/google/cadvisor -v ${PWD}:/go/src/github.com/google/cadvisor golang:1.19 make build

presubmit: lint
	@echo ">> checking go mod tidy"
	@./build/check_gotidy.sh
	@echo ">> checking file boilerplate"
	@./build/check_boilerplate.sh

lint:
	@# This assumes GOPATH/bin is in $PATH -- if not, the target will fail.
	@if ! golangci-lint version | grep $(GOLANGCI_VER); then \
		echo ">> installing golangci-lint $(GOLANGCI_VER)"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_VER); \
	fi
	@echo ">> running golangci-lint using configuration at .golangci.yml"
	@golangci-lint run
	@cd cmd && golangci-lint run

clean:
	@rm -f *.test cadvisor
	@rm -rf _output/

.PHONY: all build docker format release test test-integration lint presubmit tidy
