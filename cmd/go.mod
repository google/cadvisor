module github.com/google/cadvisor/cmd

go 1.13

// Record that the cmd module requires the cadvisor library module.
// The github.com/google/cadvisor/cmd module is built using the Makefile
// from a clone of the github.com/google/cadvisor repository, so we
// always use the relative local source rather than specifying a module version.
require github.com/google/cadvisor v0.0.0

// Use the relative local source of the github.com/google/cadvisor library to build
replace github.com/google/cadvisor => ../

require (
	github.com/Rican7/retry v0.1.1-0.20160712041035-272ad122d6e5
	github.com/SeanDolphin/bqschema v0.0.0-20150424181127-f92a08f515e1
	github.com/Shopify/sarama v1.8.0
	github.com/Shopify/toxiproxy v2.1.4+incompatible // indirect
	github.com/abbot/go-http-auth v0.0.0-20140618235127-c0ef4539dfab
	github.com/eapache/go-resiliency v1.0.1-0.20160104191539-b86b1ec0dd42 // indirect
	github.com/eapache/queue v1.0.2 // indirect
	github.com/garyburd/redigo v0.0.0-20150301180006-535138d7bcd7
	github.com/golang/snappy v0.0.0-20150730031844-723cc1e459b8 // indirect
	github.com/influxdb/influxdb v0.9.6-0.20151125225445-9eab56311373
	github.com/klauspost/crc32 v1.2.0 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/mesos/mesos-go v0.0.7-0.20180413204204-29de6ff97b48
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.7.1 // indirect
	github.com/opencontainers/runc v1.0.0-rc10
	github.com/pquerna/ffjson v0.0.0-20171002144729-d49c2bc1aa13 // indirect
	github.com/prometheus/client_golang v1.0.0
	github.com/stretchr/testify v1.4.0
	golang.org/x/oauth2 v0.0.0-20180821212333-d2e6202438be
	google.golang.org/api v0.0.0-20150730141719-0c2979aeaa5b
	google.golang.org/appengine v1.6.5 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/olivere/elastic.v2 v2.0.12
	gopkg.in/yaml.v2 v2.2.8 // indirect
	k8s.io/klog/v2 v2.0.0
	k8s.io/utils v0.0.0-20200414100711-2df71ebbae66
)
