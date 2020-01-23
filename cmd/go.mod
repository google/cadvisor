module github.com/google/cadvisor/cmd

go 1.13

require (
	github.com/google/cadvisor v0.35.0
	github.com/google/cadvisor/registry v0.35.0
	github.com/google/cadvisor/storage v0.35.0
)

replace (
	github.com/google/cadvisor => ../
	github.com/google/cadvisor/info => ../info
	github.com/google/cadvisor/registry => ../registry
	github.com/google/cadvisor/storage => ../storage
)

require (
	github.com/stretchr/testify v1.4.0
	k8s.io/klog v0.3.0
)
