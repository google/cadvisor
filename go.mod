module github.com/google/cadvisor

go 1.13

require (
	cloud.google.com/go v0.26.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.3.2 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/Rican7/retry v0.1.1-0.20160712041035-272ad122d6e5
	github.com/SeanDolphin/bqschema v0.0.0-20150424181127-f92a08f515e1
	github.com/Shopify/sarama v1.8.0
	github.com/Shopify/toxiproxy v2.1.4+incompatible // indirect
	github.com/abbot/go-http-auth v0.0.0-20140618235127-c0ef4539dfab
	github.com/aws/aws-sdk-go v1.6.10
	github.com/beorn7/perks v0.0.0-20150223135152-b965b613227f // indirect
	github.com/blang/semver v3.1.0+incompatible
	github.com/checkpoint-restore/go-criu v0.0.0-20190109184317-bdb7599cd87b // indirect
	github.com/containerd/console v0.0.0-20170925154832-84eeaae905fa // indirect
	github.com/containerd/containerd v1.3.0-beta.2.0.20190828155532-0293cbd26c69
	github.com/containerd/ttrpc v0.0.0-20191028202541-4f1b8fe65a5c // indirect
	github.com/containerd/typeurl v0.0.0-20190911142611-5eb25027c9fd
	github.com/coreos/go-systemd v0.0.0-20160527140244-4484981625c1 // indirect
	github.com/cyphar/filepath-securejoin v0.2.2-0.20170720062807-ae69057f2299 // indirect
	github.com/docker/distribution v2.6.0-rc.1.0.20170726174610-edc3ab29cdff+incompatible // indirect
	github.com/docker/docker v1.4.2-0.20180612054059-a9fbbdc8dd87
	github.com/docker/go-connections v0.3.0
	github.com/docker/go-units v0.2.1-0.20151230175859-0bbddae09c5a
	github.com/eapache/go-resiliency v1.0.1-0.20160104191539-b86b1ec0dd42 // indirect
	github.com/eapache/queue v1.0.2 // indirect
	github.com/euank/go-kmsg-parser v2.0.0+incompatible
	github.com/garyburd/redigo v0.0.0-20150301180006-535138d7bcd7
	github.com/go-ini/ini v1.9.0 // indirect
	github.com/godbus/dbus v0.0.0-20151105175453-c7fdd8b5cd55 // indirect
	github.com/gogo/protobuf v1.3.0
	github.com/golang/snappy v0.0.0-20150730031844-723cc1e459b8 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/influxdb/influxdb v0.9.6-0.20151125225445-9eab56311373
	github.com/jmespath/go-jmespath v0.0.0-20160202185014-0b12d6b521d8 // indirect
	github.com/karrick/godirwalk v1.7.5
	github.com/kevinburke/go-bindata v3.16.0+incompatible // indirect
	github.com/klauspost/crc32 v0.0.0-20151223135126-a3b15ae34567 // indirect
	github.com/kr/pretty v0.0.0-20140723054909-088c856450c0
	github.com/kr/text v0.0.0-20130911015532-6807e777504f // indirect
	github.com/mattn/go-shellwords v1.0.4-0.20180201004752-39dbbfa24bbc // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mesos/mesos-go v0.0.7-0.20180413204204-29de6ff97b48
	github.com/mindprince/gonvml v0.0.0-20171110221305-fee913ce8fb2
	github.com/mistifyio/go-zfs v2.1.2-0.20170901132433-166dd29edf05+incompatible
	github.com/mrunalp/fileutils v0.0.0-20160930181131-4ee1cc9a8058 // indirect
	github.com/onsi/ginkgo v1.10.3 // indirect
	github.com/onsi/gomega v1.7.1 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.0-rc6.0.20170604055404-372ad780f634 // indirect
	github.com/opencontainers/runc v1.0.0-rc8.0.20190906011214-a6606a7ae9d9
	github.com/opencontainers/runtime-spec v1.0.0
	github.com/opencontainers/selinux v1.3.1-0.20190929122143-5215b1806f52 // indirect
	github.com/pborman/uuid v0.0.0-20150824212802-cccd189d45f7 // indirect
	github.com/pkg/errors v0.8.1
	github.com/pquerna/ffjson v0.0.0-20171002144729-d49c2bc1aa13 // indirect
	github.com/prometheus/client_golang v0.9.1
	github.com/prometheus/client_model v0.0.0-20170216185247-6f3806018612
	github.com/prometheus/common v0.0.0-20170220103846-49fee292b27b
	github.com/seccomp/libseccomp-golang v0.0.0-20150813023252-1b506fc7c24e // indirect
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2 // indirect
	github.com/vishvananda/netlink v0.0.0-20150820014904-1e2e08e8a2dc // indirect
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
	golang.org/x/net v0.0.0-20190603091049-60506f45cf65
	golang.org/x/oauth2 v0.0.0-20180821212333-d2e6202438be
	golang.org/x/sys v0.0.0-20190522044717-8097e1b27ff5
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/api v0.0.0-20150730141719-0c2979aeaa5b
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/grpc v1.21.0
	gopkg.in/olivere/elastic.v2 v2.0.12
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/klog v0.3.0
	k8s.io/utils v0.0.0-20190712204705-3dccf664f023
)
