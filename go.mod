module github.com/google/cadvisor

go 1.13

require (
	cloud.google.com/go v0.57.0
	github.com/aws/aws-sdk-go v1.35.24
	github.com/blang/semver v3.5.1+incompatible
	github.com/containerd/containerd v1.6.0-beta.1
	github.com/containerd/containerd/api v1.6.0-beta.1
	github.com/containerd/typeurl v1.0.2
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0
	github.com/euank/go-kmsg-parser v2.0.0+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/karrick/godirwalk v1.16.1
	github.com/mindprince/gonvml v0.0.0-20190828220739-9ebdce4bb989
	github.com/mistifyio/go-zfs v2.1.2-0.20190413222219-f784269be439+incompatible
	github.com/moby/sys/mountinfo v0.4.1
	github.com/opencontainers/runc v1.0.2
	github.com/opencontainers/runtime-spec v1.0.3-0.20210326190908-1c3f411f0417
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.26.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20210825183410-e898025ed96a
	golang.org/x/sys v0.0.0-20210915083310-ed5796bab164
	google.golang.org/grpc v1.41.0
	k8s.io/klog/v2 v2.9.0
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
)
