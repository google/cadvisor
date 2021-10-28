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
	github.com/Shopify/sarama v1.19.0
	github.com/abbot/go-http-auth v0.0.0-20140618235127-c0ef4539dfab
	github.com/garyburd/redigo v0.0.0-20150301180006-535138d7bcd7
	github.com/influxdb/influxdb v0.9.6-0.20151125225445-9eab56311373
	github.com/mesos/mesos-go v0.0.7-0.20180413204204-29de6ff97b48
	github.com/pquerna/ffjson v0.0.0-20171002144729-d49c2bc1aa13 // indirect
	github.com/prometheus/client_golang v1.8.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	google.golang.org/api v0.34.0
	gopkg.in/olivere/elastic.v2 v2.0.12
	k8s.io/klog/v2 v2.30.0
	k8s.io/utils v0.0.0-20201110183641-67b214c5f920
)
