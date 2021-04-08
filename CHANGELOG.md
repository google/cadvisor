# Changelog

### 0.39.0 (2021-03-08)

- [do not initialize libipmctl package when getting an error from nvm_init()](https://github.com/google/cadvisor/pull/2723)
- [Don't fail permenantly when nvml isn't installed](https://github.com/google/cadvisor/pull/2732)
- [Update libpfm to 4.11.0](https://github.com/google/cadvisor/pull/2746)
- [Fix race between `OnDemandHousekeeping` and `housekeepingTick`](https://github.com/google/cadvisor/pull/2755)
- [Fix timeout flooding issue after containerd restart](https://github.com/google/cadvisor/pull/2749)
- [Refactor process parsing to accommodate commands with spaces + Memory cgroup is not available on some systems](https://github.com/google/cadvisor/pull/2751)
- [Switch from k8s utils/mount to moby/sys mount](https://github.com/google/cadvisor/pull/2782)
- [Support nfs in processMounts](https://github.com/google/cadvisor/pull/2787)
- [Update docker/runc and a few other dependencies](https://github.com/google/cadvisor/pull/2790)
- [Add container_blkio_device_usage metric](https://github.com/google/cadvisor/pull/2795)
- [Update heuristic for container creation time](https://github.com/google/cadvisor/pull/2800)
- [Fix incorrect CPU topology on single NUMA and multi socket platform.](https://github.com/google/cadvisor/pull/2799)
- [Added support for filesystem metrics on Docker](https://github.com/google/cadvisor/pull/2768)
- [sched_getaffinity does not return number of online CPUs](https://github.com/google/cadvisor/pull/2805)
- [Add libipmctl to the docker image.](https://github.com/google/cadvisor/pull/2674)
- [Add cgroup_memory_migrate metric](https://github.com/google/cadvisor/pull/2796)
- [bump runc to v1.0.0-rc93](https://github.com/google/cadvisor/pull/2809)
- [Fix memory stats for cgroup v2](https://github.com/google/cadvisor/pull/2810)
- [Allow gathering of stats for root cgroup on v2](https://github.com/google/cadvisor/pull/2801)
- [Remove trailing \0 from values read from ppc64le device-tree](https://github.com/google/cadvisor/pull/2811)
- [Fix oomparser regex for kernels 5.0 and higher](https://github.com/google/cadvisor/pull/2817)
- [Handling arm64: topology and online information](https://github.com/google/cadvisor/pull/2744)
- [Bump golang to 1.16](https://github.com/google/cadvisor/pull/2818)
- [Bump containerd to 1.4.4](https://github.com/google/cadvisor/pull/2826)
- [Conditionally gathering FS usage metrics](https://github.com/google/cadvisor/pull/2828)

### 0.38.8 (2021-02-18)
- [Cherrypick to v0.38 - Fix incorrect CPU topology on single NUMA and multi socket platform](https://github.com/google/cadvisor/pulls/2799)
- [Cherrypick to v0.38 - sched_getaffinity does not return number of online CPUs](https://github.com/google/cadvisor/pulls/2805)

### 0.37.5 (2021-02-18)
- [Cherrypick to v0.37 - Fix incorrect CPU topology on single NUMA and multi socket platform](https://github.com/google/cadvisor/pulls/2799)
- [Cherrypick to v0.37 - sched_getaffinity does not return number of online CPUs](https://github.com/google/cadvisor/pulls/2805)

### 0.38.7 (2021-01-13)
- [Cherrypick to v0.37: Return correct DeviceInfo from GetDirFsDevice on / path for Btrfs - Fix kubernetes issue #94335](https://github.com/google/cadvisor/pulls/2775)

### 0.37.4 (2021-01-13)
- [Cherrypick to v0.37: Return correct DeviceInfo from GetDirFsDevice on / path for Btrfs - Fix kubernetes issue #94335](https://github.com/google/cadvisor/pulls/2776)

### 0.38.6 (2020-12-9)
- [Cherrypick to v0.37: Fix timeout flooding issue after containerd restart](https://github.com/google/cadvisor/pulls/2759)

### 0.37.3 (2020-12-9)
- [Cherrypick to v0.37: Fix timeout flooding issue after containerd restart](https://github.com/google/cadvisor/pulls/2758)

### 0.38.5 (2020-11-23)
- [Cherrypick to v0.37: don't fail permenantly when nvml isn't installed](https://github.com/google/cadvisor/pulls/2735)

### 0.37.2 (2020-11-23)
- [Cherrypick to v0.37 - update docker client method](https://github.com/google/cadvisor/pulls/2734)

### 0.37.1 (2020-11-18)
- [Cherrypick to v0.37: don't fail permenantly when nvml isn't installed](https://github.com/google/cadvisor/pulls/2737)

### 0.38.4 (2020-11-12)
- [vendor: run go mod tidy](https://github.com/google/cadvisor/pulls/2731)

### 0.38.3 (2020-11-12)
- [vendor: Rollback gopkg.in/yaml.v2 to v2.2.8](https://github.com/google/cadvisor/pulls/2728)

### 0.38.2 (2020-11-10)
- [Revert mount-utils back to utils/mount](https://github.com/google/cadvisor/pulls/2726)

### 0.38.1 (2020-11-10)
- [deps: Rollback grpc from v1.33.2 to v1.27.1](https://github.com/google/cadvisor/pull/2724)
- [do not initialize libipmctl package when getting an error from nvm_init()](https://github.com/google/cadvisor/pull/2723)

### 0.38.0 (2020-11-09)

- [#1594 - chore: add storage_driver_buffer_duration in Influxdb storage docs](https://github.com/google/cadvisor/pull/1594)
- [#1924 - add hugepages info to attributes](https://github.com/google/cadvisor/pull/1924)
- [#2578 - Add perf event grouping.](https://github.com/google/cadvisor/pull/2578)
- [#2590 - Use current Docker registry](https://github.com/google/cadvisor/pull/2590)
- [#2611 - Aggregate perf metrics](https://github.com/google/cadvisor/pull/2611)
- [#2612 - Add stats to stdout storage](https://github.com/google/cadvisor/pull/2612)
- [#2618 - Update to containerd v1.4.0-beta.2 and runc v1.0.0-rc91](https://github.com/google/cadvisor/pull/2618)
- [#2621 - Memory numa stats](https://github.com/google/cadvisor/pull/2621)
- [#2627 - use Google Charts loader and not jsapi](https://github.com/google/cadvisor/pull/2627)
- [#2631 - Add entry for libpfm related tests to Makefile](https://github.com/google/cadvisor/pull/2631)
- [#2632 - Handling zeros in readPerfStat](https://github.com/google/cadvisor/pull/2632)
- [#2638 - Add stats to statsd storage](https://github.com/google/cadvisor/pull/2638)
- [#2639 - Add logs and simplify setup of raw perf events](https://github.com/google/cadvisor/pull/2639)
- [#2640 - Remove exclude guest flag from perf event attrs. ](https://github.com/google/cadvisor/pull/2640)
- [#2644 - Use perf attributes from unix lib.](https://github.com/google/cadvisor/pull/2644)
- [#2646 - Fixed https proxy issue by installing 'full' wget in Docker alpine-based build stage](https://github.com/google/cadvisor/pull/2646)
- [#2655 - Update readme to point to discuss.kubernetes.io](https://github.com/google/cadvisor/pull/2655)
- [#2659 - Fix ordering of processes table](https://github.com/google/cadvisor/pull/2659)
- [#2665 - add clean operation when watchForNewContainers/Start failed](https://github.com/google/cadvisor/pull/2665)
- [#2669 - Update release documentation and process](https://github.com/google/cadvisor/pull/2669)
- [#2676 - Fix runtime error when there are no NVM devices.](https://github.com/google/cadvisor/pull/2676)
- [#2678 - Add checking checksum of libpfm4](https://github.com/google/cadvisor/pull/2678)
- [#2679 - Fix typo in libipmctl](https://github.com/google/cadvisor/pull/2679)
- [#2682 - Add missing flag to runtime_options.md](https://github.com/google/cadvisor/pull/2682)
- [#2683 - Add flags that were not previously published](https://github.com/google/cadvisor/pull/2683)
- [#2687 - Move mount library dependency from utils/mount to mount-utils](https://github.com/google/cadvisor/pull/2687)
- [#2689 - Increase the readability of perf event logs.](https://github.com/google/cadvisor/pull/2689)
- [#2690 - Try to read from sysfs before giving up on non-x86_64](https://github.com/google/cadvisor/pull/2690)
- [#2691 - Broken build configuration when custom build tags are used](https://github.com/google/cadvisor/pull/2691)
- [#2695 - Add information about limits of opened perf event files.](https://github.com/google/cadvisor/pull/2695)
- [#2697 - Update to new docker(v19.03.13) and containerd(1.4.1)](https://github.com/google/cadvisor/pull/2697)
- [#2702 - Increase golang ci lint timeout to 5 minutes](https://github.com/google/cadvisor/pull/2702)
- [#2706 - Add a badge for the current e2e test result](https://github.com/google/cadvisor/pull/2706)
- [#2707 - Fix Avoid random values in unix.PerfEventAttr{}](https://github.com/google/cadvisor/pull/2707)
- [#2711 - validateMemoryAccounting: fix for cgroup v2](https://github.com/google/cadvisor/pull/2711)
- [#2713 - Bump golang to 1.15](https://github.com/google/cadvisor/pull/2713)
- [#2714 - update docker client method](https://github.com/google/cadvisor/pull/2714)
- [#2716 - Update dependencies](https://github.com/google/cadvisor/pull/2716)

### 0.35.1 (2020-11-05)
- [Make a copy of MachineInfo in GetMachineInfo()](https://github.com/google/cadvisor/pull/2490)

### 0.37.0 (2020-07-07)
- Add on-demand collection for prometheus metrics
- Fix detection of image filesystem
- Fix disk metrics for devicemapper devices
- Add NVM Power and NVM, Dimm, memory information to machine info
- Fix detection of OOM Kills on 5.0 linux kernels
- Add support for perf core and uncore event monitoring
- Add hugetlb container metrics
- Split into multiple go modules
- Add referenced memory metrics
- Publish images to gcr.io/cadvisor instead of gcr.io/google_containers
- Add socket id to numa topology in machine info
- Add resource control (Resctlr) metrics

### 0.36.0 (2020-02-28)
- Add support for risc and mips CPUs
- Add advanced TCP stats
- Fix bug in which cAdvisor could fail to discover docker's root directory
- The stdout storage driver now supports metric timestamps
- Add ulimit metrics
- Support multi-arch container builds
- Switch to go modules

### 0.35.0 (2019-11-27)
- Add hugepage info per-numa-node
- Add support for cgoups v2 unified higherarchy
- Drop support for rkt
- Fix a bug that prevented running with multiple tmpfs mounts

### 0.34.0 (2019-08-26)
- Fix disk stats in LXD using ZFS storage pool
- Support monitoring non-k8s containerd namespaces
- The `storage_driver` flag now supports comma-separated inputs
- Add `container_sockets`, `container_threads`, and `container_threads_max` metrics
- Fix CRI-O missing network metris bug
- Add `disable_root_cgroup_stats` flag to allow not collecting stats from the root cgroup.

### 0.33.0 (2019-02-26)
- Add --raw_cgroup_prefix_whitelist flag to allow configuring which raw cgroup trees cAdvisor monitors
- Replace `du` and `find` with a golang implementation
- Periodically update MachineInfo to support hot-add/remove
- Add explicit timestamps to prometheus metrics to fix rate calculations
- Add --url_base_prefix flag to provide better support for reverse proxies
- Add --white_listed_container_labels flag to allow specifying the container labels added as prometheus labels

### 0.32.0 (2018-11-12)
- Add container process and file descriptor metrics (disabled by default)
- Rename `type` label to `failure_type` for prometheus `memory_failures_total` metric
- Reduce mesos error logging when mesos not present

### 0.31.0 (2018-09-07)
- Fix NVML initialization race condition
- Fix brtfs filesystem discovery
- Fix race condition with AllDockerContainers
- Don't watch .mount cgroups
- Reduce lock contention during list containers
- Don't produce prometheus metrics for ignored metrics
- Add option to not export container labels as prometheus labels
- Docs: Publish cAdvisor daemonset
- Docs: Add documentation for exported prometheus metrics

### 0.30.1 (2018-06-11)
- Revert switch from inotify to fsnotify

### 0.30.0 (2018-06-05)
- Use IONice to reduce IO priority of `du` and `find`
- BREAKING API CHANGE: ContainerReference no longer contains Labels.  Use ContainerSpec instead.
- Add schedstat metrics, disabled by default.
- Fix a bug where cadvisor failed to discover a sub-cgroup that was created soon after the parent cgroup.

### 0.29.0 (2018-02-20)
- Disable per-cpu metrics by default for scalability
- Fix disk usage monitoring of overlayFs
- Retry docker connection on startup timeout

### 0.28.3 (2017-12-7)
- Add timeout for docker calls
- Fix prometheus label consistency

### 0.28.2 (2017-11-21)
- Fix GPU init race condition

### 0.28.1 (2017-11-20)
- Add containerd support
- Fix fsnotify regression from 0.28.0
- Add on demand metrics

### 0.28.0 (2017-11-06)
- Add container nvidia GPU metrics
- Expose container memory max_usage_in_bytes
- Add container memory reservation to prometheus

### 0.27.1 (2017-09-06)
- Add CRI-O support

### 0.27.0 (2017-09-01)
- Fix journalctl leak
- Fix container memory rss
- Add hugepages support
- Fix incorrect CPU usage with 4.7 kernel
- OOM parser uses kmsg
- Add tmpfs support

### 0.26.1 (2017-06-21)
- Fix prometheus metrics.

### 0.26.0 (2017-05-31)
- Fix disk partition discovery for brtfs
- Add ZFS support
- Add UDP metrics (collection disabled by default)
- Improve diskio prometheus metrics
- Update Prometheus godeps to v0.8
- Add overlay2 storage driver support

### 0.25.0 (2017-03-09)
- Disable thin_ls due to excessive iops
- Ignore .mount cgroups, fixing dissappearing stats
- Fix wc goroutine leak
- Update aws-sdk-go dependency to 1.6.10
- Update to go 1.7 for releases

### 0.24.1 (2016-10-10)

- Fix issue with running cAdvisor in a container on some distributions.

### 0.24.0 (2016-09-19)

- Added host-level inode stats (total & available)
- Improved robustness to partial failures
- Metrics collector improvements
  - Added ability to directly use endpoints from the container itself
  - Allow SSL endpoint access
  - Ability to provide a certificate which is exposed to custom endpoints
- Lots of bug fixes, including:
  - Devicemapper thin_ls fixes
  - Prometheus metrics fixes
  - Fixes for missing stats (memory reservation, FS usage, etc.)

### 0.23.9 (2016-08-09)

- Cherry-pick release:
  - Ensure minimum kernel version for thin_ls

### 0.23.8 (2016-08-02)

- Cherry-pick release:
  - Prefix Docker labels & env vars in Prometheus metrics to prevent conflicts

### 0.23.7 (2016-07-18)

- Cherry-pick release:
  - Modify working set memory stats calculation

### 0.23.6 (2016-06-23)

- Cherry-pick release:
  - Updating inotify to fix memory leak v0.23 cherrypick

### 0.23.5 (2016-06-22)

- Cherry-pick release:
  - support LVM based device mapper storage drivers

### 0.23.4 (2016-06-16)
- Cherry-pick release:
  - Check for thin_is binary in path for devicemapper when using ThinPoolWatcher
  - Fix uint64 overflow issue for CPU stats

### 0.23.3 (2016-06-08)
- Cherry-pick release:
  - Cap the maximum consecutive du commands
  - Fix a panic when a prometheus endpoint ends with a newline

### 0.23.2 (2016-05-18)
- Handle kernel log rotation
- More rkt support: poll rkt service for new containers
- Better handling of partial failures when fetching subcontainers
- Devicemapper thin_ls support (requires Device Mapper kernel module and supporting utilities)

### 0.23.1 (2016-05-11)
- Add multi-container charts to the UI
- Add TLS options for Kafka storage driver
- Switch to official Docker client
- Systemd:
  - Ignore .mount cgroups on systemd
  - Better OOM monitoring
- Bug: Fix broken -disable_metrics flag
- Bug: Fix openstack identified as AWS
- Bug: Fix EventStore when limit is 0

### 0.23.0 (2016-04-21)
- Docker v1.11 support
- Preliminary rkt support
- Bug: Fix file descriptor leak

### 0.22.0 (2016-02-25)
- Disk usage calculation bug fixes
- Systemd integration bug fixes
- Instance ID support for Azure and AWS
- Limit number of custom metrics
- Support opt out for disk and network metrics

### 0.21.0 (2016-02-03)
- Support for filesystem stats with docker v1.10
- Bug fixes.

### 0.20.5 (2016-01-27)
- Breaking: Use uint64 for memory stats
- Bug: Fix devicemapper partition labelling
- Bug: Fix network stats when using new Docker network functionality
- Bug: Fix env var label mapping initialization
- Dependencies: libcontainer update

### 0.20.4 (2016-01-20)
- Godep updates

### 0.20.3 (2016-01-19)
- Bug fixes
- Jitter added to housekeeping to smooth CPU usage.

### 0.20.2 (2016-01-15)
- New v2.1 API with better filesystem stats
- Internal refactoring
- Bug fixes.

### 0.18.0 (2015-09-23)
- Large bunch of bug-fixes
- Fixed networking stats for newer docker versions using libnetwork.
- Added application-specific metrics

## 0.16.0 (2015-06-26)
- Misc fixes.

## 0.15.1 (2015-06-10)
- Fix longstanding memory leak.
- Fix UI on newest Chrome.

## 0.15.0 (2015-06-08)
- Expose multiple network intefaces in UI and API.
- Add support for XFS.
- Fixes in inotify watches.
- Fixes on PowerPC machines.
- Fixes for newer systems with systemd.
- Extra debuging informaiton in /validate.

## 0.14.0 (2015-05-21)
- Add process stats to container pages in the UI.
- Serve UI from relative paths (allows reverse proxying).
- Minor fixes to events API.
- Add bytes available to FS info.
- Adding Docker status and image information to UI.
- Basic Redis storage backend.
- Misc reliability improvements.

## 0.13.0 (2015-05-01)
- Added `--docker_only` to limit monitoring to only Docker containers.
- Added support for Docker labels.
- Added limit for events storage.
- Fixes for OOM event monitoring.
- Changed event type to a string in the API.
- Misc fixes.

## 0.12.0 (2015-04-15)
- Added support for Docker 1.6.
- Split OOM event into OOM kill and OOM.
- Made EventData a concrete type in returned events.
- Enabled CPU load tracking (experimental).

## 0.11.0 (2015-03-27)
- Export all stats as [Prometheus](https://prometheus.io/) metrics.
- Initial support for [events](docs/api.md): creation, deletion, and OOM.
- Adding machine UUID information.
- Beta release of the cAdvisor [2.0 API](docs/api_v2.md).
- Improve handling of error conditions.
- Misc fixes and improvements.

## 0.10.1 (2015-02-27)
- Disable OOM monitoring which is using too much CPU.
- Fix break in summary stats.

## 0.10.0 (2015-02-24)
- Adding Start and End time for ContainerInfoRequest.
- Various misc fixes.

## 0.9.0 (2015-02-06)
- Support for more network devices (all non-eth).
- Support for more partition types (btrfs, device-mapper, whole-disk).
- Added reporting of DiskIO stats.
- Adding container creation time to ContainerSpec.
- More robust handling of stats failures.
- Various misc fixes.

## 0.8.0 (2015-01-09)
- Added ethernet device information.
- Added machine-wide networking statistics.
- Misc UI fixes.
- Fixes for partially-isolated containers.

## 0.7.1 (2014-12-23)
- Avoid repeated logging of container errors.
- Handle non identify mounts for cgroups.

## 0.7.0 (2014-12-18)
- Support for HTTP basic auth.
- Added /validate to perform basic checks and determine support for cAdvisor.
- All stats in the UI are now updated.
- Added gauges for filesystem usage.
- Added device information to machine info.
- Fixes to container detection.
- Fixes for systemd detection.
- ContainerSpecs are now cached.
- Performance improvements.

## 0.6.2 (2014-11-20)
- Fixes for Docker API and UI endpoints.
- Misc UI bugfixes.

## 0.6.1 (2014-11-18)
- Bug fix in InfluxDB storage driver. Container name and hostname will be exported.

## 0.6.0 (2014-11-17)
- Adding /docker UI endpoint for Docker containers.
- Fixes around handling Docker containers.
- Performance enhancements.
- Embed all external dependencies.
- ContainerStats Go struct has been flattened. The wire format remains unchanged.
- Misc bugfixes and cleanups.

## 0.5.0 (2014-10-28)
- Added disk space stats. On by default for root, available on AUFS Docker containers.
- Introduced v1.2 remote API with new "docker" resource for Docker containers.
- Added "ContainerHints" file based interface to inject extra information about containers.

## 0.4.1 (2014-09-29)
- Support for Docker containers in systemd systems.
- Adding DiskIO stats
- Misc bugfixes and cleanups

## 0.4.0 (2014-09-19)
- Various performance enhancements: brings CPU usage down 85%+
- Implemented dynamic sampling through dynamic housekeeping.
- Memory storage driver is always on, BigQuery and InfluxDB are now optional storage backends.
- Fix for DNS resolution crashes when contacting InfluxDB.
- New containers are now detected using inotify.
- Added pprof HTTP endpoint.
- UI bugfixes.

## 0.3.0 (2014-09-05)
- Support for Docker with LXC backend.
- Added BigQuery storage driver.
- Performance and stability fixes for InfluxDB storage driver.
- UI fixes and improvements.
- Configurable stats gathering interval (default: 1s).
- Improvements to startup and CPU utilization.
- Added /healthz endpoint for determining whether cAdvisor is healthy.
- Bugfixes and performance improvements.

## 0.2.2 (2014-08-13)
- Improvements to influxDB plugin.
	Table name is now 'stats'.
	Network stats added.
	Detailed cpu and memory stats are no longer exported to reduce the load on the DB.
	Docker container alias now exported - It is now possible to aggregate stats across multiple nodes.
- Make the UI independent of the storage backend by caching recent stats in memory.
- Switched to glog.
- Bugfixes and performance improvements.
- Introduced v1.1 remote API with new "subcontainers" resource.

## 0.2.1 (2014-07-25)
- Handle old Docker versions.
- UI fixes and other bugfixes.

## 0.2.0 (2014-07-24)
- Added network stats to the UI.
- Added support for CoreOS and RHEL.
- Bugfixes and reliability fixes.

## 0.1.4 (2014-07-22)
- Add network statistics to REST API.
- Add "raw" driver to handle non-Docker containers.
- Remove lmctfy in favor of the raw driver.
- Bugfixes for Docker containers and logging.

## 0.1.3 (2014-07-14)
- Add support for systemd systems.
- Fixes for UI with InfluxDB storage driver.

## 0.1.2 (2014-07-10)
- Added Storage Driver concept (flag: storage_driver), default is the in-memory driver
- Implemented InfluxDB storage driver
- Support in REST API for specifying number of stats to return
- Allow running without lmctfy (flag: allow_lmctfy)
- Bugfixes

## 0.1.0 (2014-06-14)
- Support for container aliases
- Sampling historical usage and exporting that in the REST API
- Bugfixes for UI

## 0.0.0 (2014-06-10)
- Initial version of cAdvisor
- Web UI with auto-updating stats
- v1.0 REST API with container and machine information
- Support for Docker containers
- Support for lmctfy containers
