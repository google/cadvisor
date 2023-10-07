module github.com/google/cadvisor

go 1.19

replace golang.org/x/net => golang.org/x/net v0.7.0

require (
	cloud.google.com/go/compute/metadata v0.2.3
	github.com/Microsoft/go-winio v0.4.15
	github.com/aws/aws-sdk-go v1.35.24
	github.com/blang/semver/v4 v4.0.0
	github.com/containerd/ttrpc v1.2.2
	github.com/containerd/typeurl v1.0.2
	github.com/docker/docker v20.10.21+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.5.0
	github.com/euank/go-kmsg-parser v2.0.0+incompatible
	github.com/gogo/googleapis v1.4.1
	github.com/gogo/protobuf v1.3.2
	github.com/karrick/godirwalk v1.17.0
	github.com/mistifyio/go-zfs v2.1.2-0.20190413222219-f784269be439+incompatible
	github.com/moby/sys/mountinfo v0.6.2
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/runc v1.1.9
	github.com/opencontainers/runtime-spec v1.0.3-0.20220909204839-494a5a6aca78
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.14.0
	github.com/prometheus/client_model v0.3.0
	github.com/prometheus/common v0.37.0
	github.com/stretchr/testify v1.8.2
	golang.org/x/net v0.10.0
	golang.org/x/sys v0.8.0
	google.golang.org/grpc v1.57.0
	k8s.io/klog/v2 v2.100.1
	k8s.io/utils v0.0.0-20230406110748-d93618cff8a2
)

require (
	cloud.google.com/go/compute v1.23.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/checkpoint-restore/go-criu/v5 v5.3.0 // indirect
	github.com/cilium/ebpf v0.7.0 // indirect
	github.com/containerd/console v1.0.3 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/cyphar/filepath-securejoin v0.2.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/godbus/dbus/v5 v5.0.6 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2 // indirect
	github.com/moby/term v0.0.0-20221205130635-1aeaba878587 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mrunalp/fileutils v0.5.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/opencontainers/selinux v1.10.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/seccomp/libseccomp-golang v0.9.2-0.20220502022130-f33da4d89646 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/vishvananda/netlink v1.1.0 // indirect
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
	golang.org/x/text v0.9.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230803162519-f966b187b2e5 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/v3 v3.0.2 // indirect
)
