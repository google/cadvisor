# rkt roadmap

This document defines a high level roadmap for rkt development.
The dates below should not be considered authoritative, but rather indicative of the projected timeline of the project.
The [milestones defined in GitHub](https://github.com/coreos/rkt/milestones) represent the most up-to-date state of affairs.

rkt is an implementation of the [App Container spec](https://github.com/appc/spec), which is still under active development on an approximately similar timeframe.
The version of the spec that rkt implements can be seen in the output of `rkt version`.

rkt's version 1.0 release marks the command line user interface and on-disk data structures as stable and reliable for external development. The (optional) API for pod inspection is not yet completely stabilized, but is quite usable.

### rkt 1.7 (May)

- enhanced DNS configuration [#2044](https://github.com/coreos/rkt/issues/2044)
- app-level seccomp support [#1614](https://github.com/coreos/rkt/issues/1614)

### rkt 1.8 (June)

- stable gRPC [API](https://github.com/coreos/rkt/tree/master/api/v1alpha)
- IPv6 support [appc/cni#31](https://github.com/appc/cni/issues/31)
- full integration with Kubernetes (aka "rktnetes")
- full integration with `machinectl login` and `systemd-run` [#1463](https://github.com/coreos/rkt/issues/1463)
- `rkt fly` as top-level command [#1889](https://github.com/coreos/rkt/issues/1889)
- rkt runs on Fedora with SELinux in enforcing mode
- packaged for more distributions
  - CentOS [#1305](https://github.com/coreos/rkt/issues/1305)
- user configuration for stage1 [#2013](https://github.com/coreos/rkt/issues/2013)
