# Integration Testing cAdvisor 

## Docker-based tests

The cAdvisor integration tests are run per-pr using [Github Actions](https://help.github.com/en/actions). Workflow configuration can be found at [.github/workflows/test.yml](.github/workflows/test.yml). Tests are executed in Docker containers run on MS Azure virtual machines.

To run them locally Docker must be installed on your machine. Following command allows you to execute default suite of integration tests:

```
make docker-test-integration
```

Build scripts take care of building cAdvisor and integration tests, and executing them against running cAdvisor process.

In order to run non-default tests suites (e.g. such that rely on third-party C libraries) you must source one of the files available at [build/config](build/config), e.g.:

```
source build/config/libpfm4.sh && make docker-test-integration
```

All the necessary packages will be installed, build flags will be applied and additional parameters will be passed to cAdvisor automatically. Configuration is performed using shell environment variables.

## VM-base tests (legacy)

The cAdvisor integration tests are run per-pr using the [kubernetes node-e2e testing framework](https://github.com/kubernetes/community/blob/master/contributors/devel/e2e-node-tests.md) on GCE instances.  To make use of this framework, complete the setup of GCP described in the node-e2e testing framework, clone `k8s.io/kubernetes`, and from that repository run:
```
$ make test-e2e-node TEST_SUITE=cadvisor REMOTE=true
```
This will create a VM, build cadvisor, run integration tests on that VM, retrieve logs, and will clean up the test afterwards.  See the [node-e2e testing documentation](https://github.com/kubernetes/community/blob/master/contributors/devel/e2e-node-tests.md) for more running options.

To simply run the tests against an existing cAdvisor:

```
$ go test github.com/google/cadvisor/integration/tests/... -host=HOST -port=PORT
```

Note that `HOST` and `PORT` default to `localhost` and `8080` respectively.
Today We only support remote execution in Google Compute Engine since that is where we run our continuous builds.
