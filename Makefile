all: build

deps:
	go get github.com/tools/godep

build: clean deps
	godep go build .

test: clean deps build
	godep go test ./... -test.short

clean:
	rm -f cadvisor

.PHONY: all deps build test clean
