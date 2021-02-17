# Building and Testing cAdvisor

**Note**: cAdvisor only builds on Linux since it uses Linux-only APIs.

## Installing Dependencies

cAdvisor is written in the [Go](http://golang.org) programming language. If you haven't set up a Go development environment, please follow [these instructions](http://golang.org/doc/code.html) to install go tool and set up GOPATH. Note that the version of Go in package repositories of some operating systems is outdated, so please [download](https://golang.org/dl/) the latest version.

**Note**: cAdvisor requires Go 1.14 to build.

After setting up Go, you should be able to `go get` cAdvisor as expected (we use `-d` to only download):

```
$ go get -d github.com/google/cadvisor
```

## Building from Source

At this point you can build cAdvisor from the source folder:

```
$GOPATH/src/github.com/google/cadvisor $ make build
```

or run only unit tests:

```
$GOPATH/src/github.com/google/cadvisor $ make test
```

For integration tests, see the [integration testing](integration_testing.md) page.

### Non-volatile Memory Support

cAdvisor can be linked against [libipmctl](https://github.com/intel/ipmctl) library that allows to gather information about Intel® Optane™ DC Persistent memory. If you want to build cAdvisor with libipmctl support you must meet following requirements:
* `libipmctl-devel` must be installed on build system.
* `libipmctl` must be installed on all systems where cAdvisor is running.

Detailed information about building `libipmctl` can be found in the project's [README](https://github.com/intel/ipmctl#build). Make sure to use the most up to date released version. Functionality that relies on `libipmctl` was tested against version 02.00.00.3820 of the library.

To enable `libipmctl` support `GO_FLAGS` variable must be set:

```
$GOPATH/src/github.com/google/cadvisor $ GO_FLAGS="-tags=libipmctl,netgo" make build
```

### Perf Support

cAdvisor can be linked against [libpfm4](http://perfmon2.sourceforge.net/) library that allows to gather information about performance monitoring events.
If you want to build cAdvisor with libpfm4 support you must meet following requirements:
* `libpfm4-dev` must be installed on build system.
* `libpfm4` must be installed on all systems where cAdvisor is running.

libpfm4 packages are available in Debian- and RHEL-derivatives distributions.

libpfm4 can be installed using apt package manager:
```
apt-get install libpfm4 libpfm4-dev
```
or yum package manager:
```
yum install libpfm libpfm-devel
```

To enable `libpfm4` support `GO_FLAGS` variable must be set:

```
$GOPATH/src/github.com/google/cadvisor $ GO_FLAGS="-tags=libpfm,netgo" make build
```

## Running Built Binary

Now you can run the built binary:

```
$GOPATH/src/github.com/google/cadvisor $ sudo ./cadvisor
```

### Perf Support

It is required to include perf config (examplary config is available [here](../../perf/testing/perf-non-hardware.json)) to run cAdvisor with performance monitoring events:
```
$GOPATH/src/github.com/google/cadvisor $ sudo ./cadvisor -perf_events_config=perf/testing/perf-non-hardware.json

```
