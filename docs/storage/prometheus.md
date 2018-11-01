# Monitoring cAdvisor with Prometheus

cAdvisor exposes container statistics as [Prometheus](https://prometheus.io) metrics out of the box. By default, these metrics are served under the `/metrics` HTTP endpoint. This endpoint may be customized by setting the `-prometheus_endpoint` command-line flag.

To monitor cAdvisor with Prometheus, simply configure one or more jobs in Prometheus which scrape the relevant cAdvisor processes at that metrics endpoint. For details, see Prometheus's [Configuration](https://prometheus.io/docs/operating/configuration/) documentation, as well as the [Getting started](https://prometheus.io/docs/introduction/getting_started/) guide.

# Examples

* [CenturyLink Labs](https://labs.ctl.io/) did an excellent write up on [Monitoring Docker services with Prometheus +cAdvisor](https://www.ctl.io/developers/blog/post/monitoring-docker-services-with-prometheus/), while it is great to get a better overview of cAdvisor integration with Prometheus, the PromDash GUI part is outdated as it has been deprecated for Grafana.

* [vegasbrianc](https://github.com/vegasbrianc) provides a [starter project](https://github.com/vegasbrianc/prometheus) for cAdvisor and Prometheus monitoring, alongide a ready-to-use [Grafana dashboard](https://github.com/vegasbrianc/grafana_dashboard).

## Prometheus metrics

The table below lists the Prometheus metrics exposed by cAdvisor (in alphabetical order by metric name):

Metric name | Type | Description | Unit (where applicable)
:-----------|:-----|:------------|:-----------------------
`container_accelerator_duty_cycle` | Gauge | Percent of time over the past sample period during which the accelerator was actively processing | percentage
`container_accelerator_memory_total_bytes` | Gauge | Total accelerator memory | bytes
`container_accelerator_memory_used_bytes` | Gauge | Total accelerator memory allocated | bytes
`container_cpu_cfs_periods_total` | Counter | Number of elapsed enforcement period intervals |
`container_cpu_cfs_throttled_periods_total` | Counter | Number of throttled period intervals |
`container_cpu_cfs_throttled_seconds_total` | Counter | Total time duration the container has been throttled | seconds
`container_cpu_load_average_10s` | Gauge | Value of container cpu load average over the last 10 seconds |
`container_cpu_schedstat_run_periods_total` | Counter | Number of times processes of the cgroup have run on the cpu |
`container_cpu_schedstat_run_seconds_total` | Counter | Time duration the processes of the container have run on the CPU | seconds
`container_cpu_schedstat_runqueue_seconds_total` | Counter | Time duration processes of the container have been waiting on a runqueue | seconds
`container_cpu_system_seconds_total` | Counter | Cumulative system cpu time consumed | seconds
`container_cpu_usage_seconds_total` | Counter | Cumulative cpu time consumed | seconds
`container_cpu_user_seconds_total` | Counter | Cumulative user cpu time consumed | seconds
`container_file_descriptors` | Gauge | Number of open file descriptors for the container |
`container_fs_inodes_free` | Gauge | Number of available Inodes |
`container_fs_inodes_total` | Gauge | Total number of Inodes |
`container_fs_io_current` | Gauge | Number of I/Os currently in progress |
`container_fs_io_time_seconds_total` | Counter | Cumulative count of seconds spent doing I/Os | seconds
`container_fs_io_time_weighted_seconds_total` | Counter | Cumulative weighted I/O time | seconds
`container_fs_limit_bytes` | Gauge | Number of bytes that can be consumed by the container on this filesystem | bytes
`container_fs_reads_bytes_total` | Counter | Cumulative count of bytes read | bytes
`container_fs_reads_total` | Counter | Cumulative count of reads completed |
`container_fs_read_seconds_total` | Counter | Cumulative count of seconds spent reading |
`container_fs_reads_merged_total` | Counter | Cumulative count of reads merged
`container_fs_sector_reads_total` | Counter | Cumulative count of sector reads completed
`container_fs_sector_writes_total` | Counter | Cumulative count of sector writes completed
`container_fs_usage_bytes` | Gauge | Number of bytes that are consumed by the container on this filesystem | bytes
`container_fs_write_seconds_total` | Counter | Cumulative count of seconds spent writing | seconds
`container_fs_writes_bytes_total` | Counter | Cumulative count of bytes written | bytes
`container_fs_writes_merged_total` | Counter | Cumulative count of writes merged |
`container_fs_writes_total` | Counter | Cumulative count of writes completed |
`container_last_seen` | Gauge | Last time a container was seen by the exporter | timestamp
`container_memory_cache` | Gauge | Total page cache memory | bytes
`container_memory_failcnt` | Counter | Number of memory usage hits limits |
`container_memory_failures_total` | Counter | Cumulative count of memory allocation failures |
`container_memory_max_usage_bytes` | Gauge | Maximum memory usage recorded | bytes
`container_memory_rss` | Gauge | Size of RSS | bytes
`container_memory_swap` | Gauge | Container swap usage | bytes
`container_memory_mapped_file` | Gauge | Size of memory mapped files | bytes
`container_memory_usage_bytes` | Gauge | Current memory usage, including all memory regardless of when it was accessed | bytes
`container_memory_working_set_bytes` | Gauge | Current working set | bytes
`container_network_receive_bytes_total` | Counter | Cumulative count of bytes received | bytes
`container_network_receive_packets_dropped_total` | Counter | Cumulative count of packets dropped while receiving |
`container_network_receive_packets_total` | Counter | Cumulative count of packets received |
`container_network_receive_errors_total` | Counter | Cumulative count of errors encountered while receiving |
`container_network_transmit_bytes_total` | Counter | Cumulative count of bytes transmitted | bytes
`container_network_transmit_packets_total` | Counter | Cumulative count of packets transmitted |
`container_network_transmit_packets_dropped_total` | Counter | Cumulative count of packets dropped while transmitting |
`container_network_transmit_errors_total` | Counter | Cumulative count of errors encountered while transmitting |
`container_network_tcp_usage_total` | Gauge | tcp connection usage statistic for container |
`container_network_udp_usage_total` | Gauge | udp connection usage statistic for container |
`container_processes` | Gauge | Number of processes running inside the container |
`container_spec_cpu_period` | Gauge | CPU period of the container |
`container_spec_cpu_quota` | Gauge | CPU quota of the container |
`container_spec_cpu_shares` | Gauge | CPU share of the container |
`container_spec_memory_limit_bytes` | Gauge | Memory limit for the container | bytes
`container_spec_memory_swap_limit_bytes` | Gauge | Memory swap limit for the container | bytes
`container_spec_memory_reservation_limit_bytes` | Gauge | Memory reservation limit for the container | bytes
`container_start_time_seconds` | Gauge | Start time of the container since unix epoch | seconds
`container_tasks_state` | Gauge | Number of tasks in given state (`sleeping`, `running`, `stopped`, `uninterruptible`, or `ioawaiting`) |
