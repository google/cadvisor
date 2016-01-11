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
docker_tag = $(shell build/what-tag.sh)
pkgs = $(shell $(GO) list ./...)

all: format build test

deps:
	go get github.com/tools/godep

test: deps
	@echo ">> running tests"
	@$(GO) test -short $(pkgs)

format: deps
	@echo ">> formatting code"
	@$(GO) fmt $(pkgs)

vet:
	@echo ">> vetting code"
	@$(GO) vet $(pkgs)

build: deps
	@echo ">> building binaries"
	@./build/build.sh

release: build
	@./build/release.sh

docker-build: all
	@docker build -t google/cadvisor:$(docker_tag) .

docker-push: docker-build
	@docker push google/cadvisor:$(docker_tag)

.PHONY: all format build test vet docker-build docker-push deps
