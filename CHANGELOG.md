# Changelog

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
