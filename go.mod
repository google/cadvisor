module github.com/google/cadvisor

go 1.13

require (
	github.com/google/cadvisor/info v0.35.0
	github.com/google/cadvisor/registry v0.35.0
)

replace (
	github.com/google/cadvisor/info => ./info
	github.com/google/cadvisor/registry => ./registry
)

require (
	cloud.google.com/go v0.26.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5 // indirect
	github.com/Rican7/retry v0.1.1-0.20160712041035-272ad122d6e5
	github.com/abbot/go-http-auth v0.0.0-20140618235127-c0ef4539dfab
	github.com/aws/aws-sdk-go v1.6.10
	github.com/beorn7/perks v0.0.0-20150223135152-b965b613227f // indirect
	github.com/blang/semver v3.1.0+incompatible
	github.com/checkpoint-restore/go-criu v0.0.0-20190109184317-bdb7599cd87b // indirect
	github.com/containerd/console v0.0.0-20180822173158-c12b1e7919c1 // indirect
	github.com/containerd/containerd v1.3.2
	github.com/containerd/ttrpc v0.0.0-20191028202541-4f1b8fe65a5c // indirect
	github.com/containerd/typeurl v0.0.0-20190911142611-5eb25027c9fd
	github.com/coreos/go-systemd v0.0.0-20190321100706-95778dfbb74e // indirect
	github.com/cyphar/filepath-securejoin v0.2.2-0.20170720062807-ae69057f2299 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/docker/go-connections v0.3.0
	github.com/docker/go-units v0.4.0
	github.com/euank/go-kmsg-parser v2.0.0+incompatible
	github.com/go-ini/ini v1.9.0 // indirect
	github.com/godbus/dbus v0.0.0-20190422162347-ade71ed3457e // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/jmespath/go-jmespath v0.0.0-20160202185014-0b12d6b521d8 // indirect
	github.com/karrick/godirwalk v1.7.5
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kr/pretty v0.0.0-20140723054909-088c856450c0
	github.com/kr/text v0.0.0-20130911015532-6807e777504f // indirect
	github.com/mattn/go-shellwords v1.0.4-0.20180201004752-39dbbfa24bbc // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mesos/mesos-go v0.0.7-0.20180413204204-29de6ff97b48
	github.com/mindprince/gonvml v0.0.0-20171110221305-fee913ce8fb2
	github.com/mistifyio/go-zfs v2.1.2-0.20190413222219-f784269be439+incompatible
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mrunalp/fileutils v0.0.0-20160930181131-4ee1cc9a8058 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v1.0.0-rc8.0.20190906011214-a6606a7ae9d9
	github.com/opencontainers/runtime-spec v1.0.1
	github.com/opencontainers/selinux v1.3.1-0.20190929122143-5215b1806f52 // indirect
	github.com/pkg/errors v0.8.1
	github.com/pquerna/ffjson v0.0.0-20171002144729-d49c2bc1aa13 // indirect
	github.com/prometheus/client_golang v0.9.1
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.0.0-20170220103846-49fee292b27b
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/seccomp/libseccomp-golang v0.0.0-20150813023252-1b506fc7c24e // indirect
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2 // indirect
	github.com/vishvananda/netlink v0.0.0-20150820014904-1e2e08e8a2dc // indirect
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553
	golang.org/x/sys v0.0.0-20200107162124-548cf772de50
	golang.org/x/text v0.3.2 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/genproto v0.0.0-20191230161307-f3c370f40bfb // indirect
	google.golang.org/grpc v1.26.0
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/klog v0.3.0
	k8s.io/utils v0.0.0-20200122174043-1e243dd1a584
)
