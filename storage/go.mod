module github.com/google/cadvisor/storage

go 1.13

require (
	github.com/google/cadvisor/info v0.35.0
	github.com/google/cadvisor/registry v0.35.0
)

replace (
	github.com/google/cadvisor/info => ../info
	github.com/google/cadvisor/registry => ../registry
)

require (
	github.com/SeanDolphin/bqschema v0.0.0-20150424181127-f92a08f515e1
	github.com/Shopify/sarama v1.8.0
	github.com/Shopify/toxiproxy v2.1.4+incompatible // indirect
	github.com/eapache/go-resiliency v1.0.1-0.20160104191539-b86b1ec0dd42 // indirect
	github.com/eapache/queue v1.0.2 // indirect
	github.com/garyburd/redigo v0.0.0-20150301180006-535138d7bcd7
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.0-20150730031844-723cc1e459b8 // indirect
	github.com/influxdb/influxdb v0.9.6-0.20151125225445-9eab56311373
	github.com/klauspost/crc32 v0.0.0-20151223135126-a3b15ae34567 // indirect
	github.com/onsi/ginkgo v1.10.3 // indirect
	github.com/onsi/gomega v1.7.1 // indirect
	github.com/stretchr/objx v0.1.1 // indirect
	github.com/stretchr/testify v1.4.0
	golang.org/x/net v0.0.0-20191209160850-c0dbc17a3553 // indirect
	golang.org/x/oauth2 v0.0.0-20180821212333-d2e6202438be
	golang.org/x/sys v0.0.0-20200107162124-548cf772de50 // indirect
	google.golang.org/api v0.0.0-20150730141719-0c2979aeaa5b
	google.golang.org/appengine v1.6.5 // indirect
	gopkg.in/olivere/elastic.v2 v2.0.12
	k8s.io/klog v0.3.0
)
