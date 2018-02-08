# Integration Testing cAdvisor

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
