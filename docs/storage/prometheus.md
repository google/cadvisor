# Monitoring cAdvisor with Prometheus

cAdvisor exposes container and hardware statistics as [Prometheus](https://prometheus.io) metrics out of the box. By default, these metrics are served under the `/metrics` HTTP endpoint. This endpoint may be customized by setting the `-prometheus_endpoint` and `-disable_metrics` command-line flags.

To collect some of metrics it is required to build cAdvisor with additional flags, for details see [build instructions](../development/build.md), additional flags are indicated in "additional build flag" column in table below.

To monitor cAdvisor with Prometheus, simply configure one or more jobs in Prometheus which scrape the relevant cAdvisor processes at that metrics endpoint. For details, see Prometheus's [Configuration](https://prometheus.io/docs/operating/configuration/) documentation, as well as the [Getting started](https://prometheus.io/docs/introduction/getting_started/) guide.

# Examples

* [CenturyLink Labs](https://labs.ctl.io/) did an excellent write up on [Monitoring Docker services with Prometheus +cAdvisor](https://www.ctl.io/developers/blog/post/monitoring-docker-services-with-prometheus/), while it is great to get a better overview of cAdvisor integration with Prometheus, the PromDash GUI part is outdated as it has been deprecated for Grafana.

* [vegasbrianc](https://github.com/vegasbrianc) provides a [starter project](https://github.com/vegasbrianc/prometheus) for cAdvisor and Prometheus monitoring, alongide a ready-to-use [Grafana dashboard](https://github.com/vegasbrianc/grafana_dashboard).

## Prometheus container metrics

The table below lists the Prometheus container metrics exposed by cAdvisor (in alphabetical order by metric name):

Metric name | Type | Description | Unit (where applicable) | -disable_metrics parameter | additional build flag |
:-----------|:-----|:------------|:------------------------|:---------------------------|:----------------------
`container_accelerator_duty_cycle` | Gauge | Percent of time over the past sample period during which the accelerator was actively processing | percentage | accelerator |
`container_accelerator_memory_total_bytes` | Gauge | Total accelerator memory | bytes | accelerator |
`container_accelerator_memory_used_bytes` | Gauge | Total accelerator memory allocated | bytes | accelerator |
`container_blkio_device_usage_total` | Counter | Blkio device bytes usage | bytes | diskIO | 
`container_cpu_cfs_periods_total` | Counter | Number of elapsed enforcement period intervals | | |
`container_cpu_cfs_throttled_periods_total` | Counter | Number of throttled period intervals | | |
`container_cpu_cfs_throttled_seconds_total` | Counter | Total time duration the container has been throttled | seconds | |
`container_cpu_load_average_10s` | Gauge | Value of container cpu load average over the last 10 seconds | | |
`container_cpu_schedstat_run_periods_total` | Counter | Number of times processes of the cgroup have run on the cpu | | sched |
`container_cpu_schedstat_run_seconds_total` | Counter | Time duration the processes of the container have run on the CPU | seconds | sched |
`container_cpu_schedstat_runqueue_seconds_total` | Counter | Time duration processes of the container have been waiting on a runqueue | seconds | sched |
`container_cpu_system_seconds_total` | Counter | Cumulative system cpu time consumed | seconds | |
`container_cpu_usage_seconds_total` | Counter | Cumulative cpu time consumed | seconds | |
`container_cpu_user_seconds_total` | Counter | Cumulative user cpu time consumed | seconds | |
`container_file_descriptors` | Gauge | Number of open file descriptors for the container | | process |
`container_fs_inodes_free` | Gauge | Number of available Inodes | | disk |
`container_fs_inodes_total` | Gauge | Total number of Inodes | | disk |
`container_fs_io_current` | Gauge | Number of I/Os currently in progress | | diskIO |
`container_fs_io_time_seconds_total` | Counter | Cumulative count of seconds spent doing I/Os | seconds | diskIO |
`container_fs_io_time_weighted_seconds_total` | Counter | Cumulative weighted I/O time | seconds | diskIO |
`container_fs_limit_bytes` | Gauge | Number of bytes that can be consumed by the container on this filesystem | bytes | disk |
`container_fs_reads_bytes_total` | Counter | Cumulative count of bytes read | bytes | diskIO |
`container_fs_reads_total` | Counter | Cumulative count of reads completed | | diskIO |
`container_fs_read_seconds_total` | Counter | Cumulative count of seconds spent reading | | diskIO |
`container_fs_reads_merged_total` | Counter | Cumulative count of reads merged | | diskIO |
`container_fs_sector_reads_total` | Counter | Cumulative count of sector reads completed | | diskIO |
`container_fs_sector_writes_total` | Counter | Cumulative count of sector writes completed | | diskIO |
`container_fs_usage_bytes` | Gauge | Number of bytes that are consumed by the container on this filesystem | bytes | disk |
`container_fs_write_seconds_total` | Counter | Cumulative count of seconds spent writing | seconds | diskIO |
`container_fs_writes_bytes_total` | Counter | Cumulative count of bytes written | bytes | diskIO |
`container_fs_writes_merged_total` | Counter | Cumulative count of writes merged | | diskIO |
`container_fs_writes_total` | Counter | Cumulative count of writes completed | | diskIO |
`container_hugetlb_failcnt` | Counter | Number of hugepage usage hits limits | | hugetlb |
`container_hugetlb_max_usage_bytes` | Gauge | Maximum hugepage usages recorded | bytes | hugetlb |
`container_hugetlb_usage_bytes` | Gauge | Current hugepage usage | bytes | hugetlb |
`container_last_seen` | Gauge | Last time a container was seen by the exporter | timestamp | |
`container_llc_occupancy_bytes` | Gauge | Last level cache usage statistics for container counted with RDT Memory Bandwidth Monitoring (MBM). | bytes | resctrl |
`container_memory_bandwidth_bytes` | Gauge | Total memory bandwidth usage statistics for container counted with RDT Memory Bandwidth Monitoring (MBM). | bytes | resctrl |
`container_memory_bandwidth_local_bytes` | Gauge | Local memory bandwidth usage statistics for container counted with RDT Memory Bandwidth Monitoring (MBM). | bytes | resctrl |
`container_memory_cache` | Gauge | Total page cache memory | bytes | |
`container_memory_failcnt` | Counter | Number of memory usage hits limits | | |
`container_memory_failures_total` | Counter | Cumulative count of memory allocation failures | | |
`container_memory_numa_pages` | Gauge | Number of used pages per NUMA node | | memory_numa |
`container_memory_max_usage_bytes` | Gauge | Maximum memory usage recorded | bytes | |
`container_memory_rss` | Gauge | Size of RSS | bytes | |
`container_memory_swap` | Gauge | Container swap usage | bytes | |
`container_memory_mapped_file` | Gauge | Size of memory mapped files | bytes | |
`container_memory_migrate` | Gauge | Memory migrate status | | cpuset |
`container_memory_usage_bytes` | Gauge | Current memory usage, including all memory regardless of when it was accessed | bytes | |
`container_memory_working_set_bytes` | Gauge | Current working set | bytes | |
`container_network_receive_bytes_total` | Counter | Cumulative count of bytes received | bytes | network |
`container_network_receive_packets_dropped_total` | Counter | Cumulative count of packets dropped while receiving | | network |
`container_network_receive_packets_total` | Counter | Cumulative count of packets received | | network |
`container_network_receive_errors_total` | Counter | Cumulative count of errors encountered while receiving | | network |
`container_network_transmit_bytes_total` | Counter | Cumulative count of bytes transmitted | bytes | network |
`container_network_transmit_packets_total` | Counter | Cumulative count of packets transmitted | | network |
`container_network_transmit_packets_dropped_total` | Counter | Cumulative count of packets dropped while transmitting | | network |
`container_network_transmit_errors_total` | Counter | Cumulative count of errors encountered while transmitting | | network |
`container_network_tcp_usage_total` | Gauge | tcp connection usage statistic for container | | tcp |
`container_network_tcp6_usage_total` | Gauge | tcp6 connection usage statistic for container | | tcp |
`container_network_udp_usage_total` | Gauge | udp connection usage statistic for container | | udp |
`container_network_udp6_usage_total` | Gauge | udp6 connection usage statistic for container | | udp |
`container_perf_events_total` | Counter | Scaled counter of perf core event (event can be identified by `event` label and `cpu` indicates the core for which event was measured). See [perf event configuration](../runtime_options.md#perf-events). | | | libpfm
`container_perf_metric_scaling_ratio` | Gauge | Scaling ratio for perf event counter (event can be identified by `event` label and `cpu` indicates the core for which event was measured). See [perf event configuration](../runtime_options.md#perf-events). | | | libpfm
`container_processes` | Gauge | Number of processes running inside the container | | process |
`container_referenced_bytes` | Gauge |  Container referenced bytes during last measurements cycle based on Referenced field in /proc/smaps file, with /proc/PIDs/clear_refs set to 1 after defined number of cycles configured through `referenced_reset_interval` cAdvisor parameter.</br>Warning: this is intrusive collection because can influence kernel page reclaim policy and add latency. Refer to https://github.com/brendangregg/wss#wsspl-referenced-page-flag for more details. | bytes | referenced_memory |
`container_spec_cpu_period` | Gauge | CPU period of the container | | |
`container_spec_cpu_quota` | Gauge | CPU quota of the container | | |
`container_spec_cpu_shares` | Gauge | CPU share of the container | | |
`container_spec_memory_limit_bytes` | Gauge | Memory limit for the container | bytes | |
`container_spec_memory_swap_limit_bytes` | Gauge | Memory swap limit for the container | bytes | |
`container_spec_memory_reservation_limit_bytes` | Gauge | Memory reservation limit for the container | bytes | |
`container_start_time_seconds` | Gauge | Start time of the container since unix epoch | seconds | |
`container_tasks_state` | Gauge | Number of tasks in given state (`sleeping`, `running`, `stopped`, `uninterruptible`, or `ioawaiting`) | | |
`container_perf_uncore_events_total` | Counter | Scaled counter of perf uncore event (event can be identified by `event` label, `pmu` and `socket` lables indicate the PMU and the CPU socket for which event was measured). See [perf event configuration](../runtime_options.md#perf-events)). Metric exists only for main cgroup (id="/").| | | libpfm
`container_perf_uncore_events_scaling_ratio` | Gauge | Scaling ratio for perf uncore event counter (event can be identified by `event` label, `pmu` and `socket` lables indicate the PMU and the CPU socket for which event was measured). See [perf event configuration](../runtime_options.md#perf-events). Metric exists only for main cgroup (id="/"). | | | libpfm

## Prometheus hardware metrics

The table below lists the Prometheus hardware metrics exposed by cAdvisor (in alphabetical order by metric name):

Metric name | Type | Description | Unit (where applicable) | -disable_metrics parameter | addional build flag |
:-----------|:-----|:------------|:------------------------|:---------------------------|:--------------------
`machine_cpu_cache_capacity_bytes` | Gauge |  Cache size in bytes assigned to NUMA node and CPU core | bytes | cpu_topology |
`machine_cpu_cores` | Gauge | Number of logical CPU cores | | |
`machine_cpu_physical_cores` | Gauge | Number of physical CPU cores | | |
`machine_cpu_sockets` | Gauge | Number of CPU sockets | | |
`machine_dimm_capacity_bytes` | Gauge | Total RAM DIMM capacity (all types memory modules) value labeled by dimm type,<br>information is retrieved from sysfs edac per-DIMM API (/sys/devices/system/edac/mc/) introduced in kernel 3.6 | bytes | | |
`machine_dimm_count` | Gauge | Number of RAM DIMM (all types memory modules) value labeled by dimm type,<br>information is retrieved from sysfs edac per-DIMM API (/sys/devices/system/edac/mc/) introduced in kernel 3.6 | | |
`machine_memory_bytes` | Gauge | Amount of memory installed on the machine | bytes | |
`machine_node_hugepages_count` | Gauge |  Numer of hugepages assigned to NUMA node | | cpu_topology |
`machine_node_memory_capacity_bytes` | Gauge |  Amount of memory assigned to NUMA node | bytes | cpu_topology |
`machine_nvm_avg_power_budget_watts` | Gauge |  NVM power budget | watts | | libipmctl
`machine_nvm_capacity` | Gauge | NVM capacity value labeled by NVM mode (memory mode or app direct mode) | bytes | | libipmctl
`machine_thread_siblings_count` | Gauge | Number of CPU thread siblings | | cpu_topology |