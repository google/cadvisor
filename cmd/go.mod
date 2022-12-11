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
	github.com/Rican7/retry v0.3.1
	github.com/SeanDolphin/bqschema v1.0.0
	github.com/Shopify/sarama v1.37.2
	github.com/abbot/go-http-auth v0.4.0
	github.com/garyburd/redigo v1.6.4
	github.com/influxdb/influxdb v0.9.6-0.20151125225445-9eab56311373
	github.com/mesos/mesos-go v0.0.11
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/onsi/gomega v1.24.1 // indirect
	github.com/pquerna/ffjson v0.0.0-20190930134022-aa0246cd15f7 // indirect
	github.com/prometheus/client_golang v1.14.0
	github.com/stretchr/testify v1.8.1
	golang.org/x/oauth2 v0.3.0
	google.golang.org/api v0.104.0
	gopkg.in/olivere/elastic.v2 v2.0.61
	k8s.io/klog/v2 v2.80.1
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
)
