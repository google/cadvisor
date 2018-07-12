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
`container_last_seen` | Gauge | Last time a container was seen by the exporter | timestamp
`container_memory_cache` | Gauge | Total page cache memory | bytes
`container_memory_rss` | Gauge | Size of RSS | bytes
`container_tasks_state` | Gauge | Number of tasks in given state (`sleeping`, `running`, `stopped`, `uninterruptible`, or `ioawaiting`) |
