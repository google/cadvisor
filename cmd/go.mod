module github.com/google/cadvisor/cmd

go 1.16

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
	github.com/Shopify/sarama v1.19.0
	github.com/Shopify/toxiproxy v2.1.4+incompatible // indirect
	github.com/abbot/go-http-auth v0.0.0-20140618235127-c0ef4539dfab
	github.com/eapache/go-resiliency v1.1.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/garyburd/redigo v0.0.0-20150301180006-535138d7bcd7
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db // indirect
	github.com/influxdb/influxdb v0.9.6-0.20151125225445-9eab56311373
	github.com/mesos/mesos-go v0.0.7-0.20180413204204-29de6ff97b48
	github.com/onsi/ginkgo v1.7.0 // indirect
	github.com/onsi/gomega v1.4.3 // indirect
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/pquerna/ffjson v0.0.0-20171002144729-d49c2bc1aa13 // indirect
	github.com/prometheus/client_golang v1.13.0
	github.com/rcrowley/go-metrics v0.0.0-20181016184325-3113b8401b8a // indirect
	github.com/stretchr/testify v1.8.0
	golang.org/x/oauth2 v0.0.0-20220223155221-ee480838109b
	google.golang.org/api v0.34.0
	gopkg.in/olivere/elastic.v2 v2.0.12
	k8s.io/klog/v2 v2.80.1
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
)
