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

At this point you can build cAdvisor:

```
$ godep go build github.com/google/cadvisor
```

or run unit tests:

```
$ godep go test github.com/google/cadvisor/...
```
