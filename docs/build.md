# Building and Testing cAdvisor

**Note**: cAdvisor only builds on Linux since it uses Linux-only APIs.

You should be able to `go get` cAdvisor as expected (we use `-d` to only download):

```
$ go get -d github.com/google/cadvisor
```

We use `godep` so you will need to get that as well:

```
$ go get github.com/tools/godep
```

At this point you can build cAdvisor from the source folder:

```
$GOPATH/src/github.com/google/cadvisor $ godep go build .
```

or run only unit tests:

```
$GOPATH/src/github.com/google/cadvisor $ godep go test ./... -test.short
```

For integration tests, see the [integration testing](integration_testing.md) page.

Now you can run the built binary:

```
$GOPATH/src/github.com/google/cadvisor $ sudo ./cadvisor
```
