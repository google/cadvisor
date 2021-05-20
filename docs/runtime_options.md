# cAdvisor Runtime Options

This document describes a set of runtime flags available in cAdvisor.

## Container labels
* `--store_container_labels=false` - do not convert container labels and environment variables into labels on prometheus metrics for each container.
* `--whitelisted_container_labels` - comma separated list of container labels to be converted to labels on prometheus metrics for each container. `store_container_labels` must be set to false for this to take effect.

## Limiting which containers are monitored 
* `--docker_only=false` - do not report raw cgroup metrics, except the root cgroup.
* `--raw_cgroup_prefix_whitelist` - a comma-separated list of cgroup path prefix that needs to be collected even when `--docker_only` is specified
* `--disable_root_cgroup_stats=false` - disable collecting root Cgroup stats.

## Container Hints

Container hints are a way to pass extra information about a container to cAdvisor. In this way cAdvisor can augment the stats it gathers. For more information on the container hints format see its [definition](../container/common/container_hints.go). Note that container hints are only used by the raw container driver today.

```
--container_hints="/etc/cadvisor/container_hints.json": location of the container hints file
```

## CPU

```
--enable_load_reader=false: Whether to enable cpu load reader
--max_procs=0: max number of CPUs that can be used simultaneously. Less than 1 for default (number of cores).
```

## Debugging and Logging

cAdvisor-native flags that help in debugging:

```
--log_backtrace_at="": when logging hits line file:N, emit a stack trace
--log_cadvisor_usage=false: Whether to log the usage of the cAdvisor container
--version=false: print cAdvisor version and exit
--profiling=false: Enable profiling via web interface host:port/debug/pprof/
```

From [glog](https://github.com/golang/glog) here are some flags we find useful:

```
--log_dir="": If non-empty, write log files in this directory
--logtostderr=false: log to standard error instead of files
--alsologtostderr=false: log to standard error as well as files
--stderrthreshold=0: logs at or above this threshold go to stderr
--v=0: log level for V logs
--vmodule=: comma-separated list of pattern=N settings for file-filtered logging
```

## Docker

```
--docker="unix:///var/run/docker.sock": docker endpoint (default "unix:///var/run/docker.sock")
--docker_env_metadata_whitelist="": a comma-separated list of environment variable keys that needs to be collected for docker containers
--docker_root="/var/lib/docker": DEPRECATED: docker root is read from docker info (this is a fallback, default: /var/lib/docker) (default "/var/lib/docker")
--docker-tls: use TLS to connect to docker
--docker-tls-cert="cert.pem": client certificate for TLS-connection with docker
--docker-tls-key="key.pem": private key for TLS-connection with docker
--docker-tls-ca="ca.pem": trusted CA for TLS-connection with docker
```

## Housekeeping

Housekeeping is the periodic actions cAdvisor takes. During these actions, cAdvisor will gather container stats. These flags control how and when cAdvisor performs housekeeping.

#### Dynamic Housekeeping

Dynamic housekeeping intervals let cAdvisor vary how often it gathers stats.
It does this depending on how active the container is. Turning this off
provides predictable housekeeping intervals, but increases the resource usage
of cAdvisor.

```
--allow_dynamic_housekeeping=true: Whether to allow the housekeeping interval to be dynamic
```

#### Housekeeping Intervals

Intervals for housekeeping. cAdvisor has two housekeepings: global and per-container.

Global housekeeping is a singular housekeeping done once in cAdvisor. This typically does detection of new containers. Today, cAdvisor discovers new containers with kernel events so this global housekeeping is mostly used as backup in the case that there are any missed events.

Per-container housekeeping is run once on each container cAdvisor tracks. This typically gets container stats.

```
--global_housekeeping_interval=1m0s: Interval between global housekeepings
--housekeeping_interval=1s: Interval between container housekeepings
--max_housekeeping_interval=1m0s: Largest interval to allow between container housekeepings (default 1m0s)
```

## HTTP

Specify where cAdvisor listens.

```
--http_auth_file="": HTTP auth file for the web UI
--http_auth_realm="localhost": HTTP auth realm for the web UI (default "localhost")
--http_digest_file="": HTTP digest file for the web UI
--http_digest_realm="localhost": HTTP digest file for the web UI (default "localhost")
--listen_ip="": IP to listen on, defaults to all IPs
--port=8080: port to listen (default 8080)
--url_base_prefix=/: optional path prefix aded to all resource URLs; useful when running cAdvisor behind a proxy. (default /)
```

## Local Storage Duration

cAdvisor stores the latest historical data in memory. How long of a history it stores can be configured with the `--storage_duration` flag.

```
--storage_duration=2m0s: How long to store data.
```

## Machine

```
--boot_id_file="/proc/sys/kernel/random/boot_id": Comma-separated list of files to check for boot-id. Use the first one that exists. (default "/proc/sys/kernel/random/boot_id")
--machine_id_file="/etc/machine-id,/var/lib/dbus/machine-id": Comma-separated list of files to check for machine-id. Use the first one that exists. (default "/etc/machine-id,/var/lib/dbus/machine-id")
--update_machine_info_interval=5m: Interval between machine info updates. (default 5m)
```

## Metrics

```
--application_metrics_count_limit=100: Max number of application metrics to store (per container) (default 100)
--collector_cert="": Collector's certificate, exposed to endpoints for certificate based authentication.
--collector_key="": Key for the collector's certificate
--disable_metrics=referenced_memory,cpu_topology,resctrl,udp,advtcp,sched,hugetlb,memory_numa,tcp,process: comma-separated list of metrics to be disabled. Options are 'accelerator', 'cpu_topology','disk', 'diskIO', 'memory_numa', 'network', 'tcp', 'udp', 'percpu', 'sched', 'process', 'hugetlb', 'referenced_memory', 'resctrl', 'cpuset'. (default referenced_memory,cpu_topology,resctrl,udp,advtcp,sched,hugetlb,memory_numa,tcp,process)
--prometheus_endpoint="/metrics": Endpoint to expose Prometheus metrics on (default "/metrics")
--disable_root_cgroup_stats=false: Disable collecting root Cgroup stats
```

## Storage Drivers

```
--storage_driver="": Storage driver to use. Data is always cached shortly in memory, this controls where data is pushed besides the local cache. Empty means none. Options are: <empty>, bigquery, elasticsearch, influxdb, kafka, redis, statsd, stdout
--storage_driver_buffer_duration="1m0s": Writes in the storage driver will be buffered for this duration, and committed to the non memory backends as a single transaction (default 1m0s)
--storage_driver_db="cadvisor": database name (default "cadvisor")
--storage_driver_host="localhost:8086": database host:port (default "localhost:8086")
--storage_driver_password="root": database password (default "root")
--storage_driver_secure=false: use secure connection with database
--storage_driver_table="stats": table name (default "stats")
--storage_driver_user="root": database username (default "root")
```

## Perf Events

```
--perf_events_config="" Path to a JSON file containing configuration of perf events to measure. Empty value disables perf events measuring.
```

Core perf events can be exposed on Prometheus endpoint per CPU or aggregated by event. It is controlled through `--disable_metrics` parameter with option `percpu`, e.g.:
- `--disable_metrics="percpu"` - core perf events are aggregated
- `--disable_metrics=""` - core perf events are exposed per CPU.

It's possible to get "too many opened files" error when a lot of perf events are exposed per CPU. This happens because of passing system limits.
Try to increase max number of file desctriptors with `ulimit -n <value>`.

Aggregated form of core perf events significantly decrease volume of data. For aggregated form of core perf events scaling ratio (`container_perf_metric_scaling ratio`) indicates the lowest value of scaling ratio for specific event to show the worst precision.

### Perf subsystem introduction

One of the goals of kernel perf subsystem is to instrument CPU performance counters that allow to profile applications.
Profiling is performed by setting up performance counters that count hardware events (e.g. number of retired
instructions, number of cache misses). The counters are CPU hardware registers and amount of them is limited.

Other goals of perf subsystem (such as tracing) are beyond the scope of this documentation and you can follow Further
Reading section below to learn more about them.

Familiarize yourself with following perf-event-related terms:
* `multiplexing` - 2nd Generation Intel® Xeon® Scalable Processors provides 4 counters per each hyper thread. If number
of configured events is greater than number of available counters then Linux will multiplex counting and some (or even
all) of the events will not be accounted for all the time. In such situation information about amount of time that event
was accounted for and amount of time when event was enabled is provided. Counter value that cAdvisor exposes is scaled
automatically.
* `grouping` - in scenario when accounted for events are used to calculate derivative metrics, it is reasonable to
measure them in transactional manner: all the events in a group must be accounted for in the same period of time. Keep
in mind that it is impossible to group more events that there are counters available.
* `uncore events` - events which can be counted by PMUs outside core.
* `PMU` - Performance Monitoring Unit

#### Getting config values
Using perf tools:
* Identify the event in `perf list` output. 
* Execute command: `perf stat -I 5000 -vvv -e EVENT_NAME`
* Find `perf_event_attr` section on `perf stat` output, copy config and type field to configuration file.

```
------------------------------------------------------------
perf_event_attr:
  type                             18
  size                             112
  config                           0x304
  sample_type                      IDENTIFIER
  read_format                      TOTAL_TIME_ENABLED|TOTAL_TIME_RUNNING
  disabled                         1
  inherit                          1
  exclude_guest                    1
------------------------------------------------------------
```
* Configuration file should look like: 
```json
{
  "core": {
    "events": [
      "event_name"
    ],
    "custom_events": [
      {
        "type": 18,
        "config": [
          "0x304"
        ],
        "name": "event_name"
      }
    ]
  },
  "uncore": {
    "events": [
      "event_name"
    ],
    "custom_events": [
      {
        "type": 18,
        "config": [
          "0x304"
        ],
        "name": "event_name"
      }
    ]
  }
}
```

Config values can be also obtain from: 
* [Intel® 64 and IA32 Architectures Performance Monitoring Events](https://software.intel.com/content/www/us/en/develop/download/intel-64-and-ia32-architectures-performance-monitoring-events.html)


##### Uncore Events configuration
Uncore Event name should be in form `PMU_PREFIX/event_name` where **PMU_PREFIX** mean
that statistics would be counted on all PMUs with that prefix in name.

Let's explain this by example: 

```json
{
  "uncore": {
    "events": [
      "uncore_imc/cas_count_read",
      "uncore_imc_0/cas_count_write",
      "cas_count_all"
    ],
    "custom_events": [ 
      {
        "config": [
          "0x304"
        ],
        "name": "uncore_imc_0/cas_count_write"
      },
      {
        "type": 19,
        "config": [
          "0x304"
        ],
        "name": "cas_count_all"
      }
    ]
  }
}
```

- `uncore_imc/cas_count_read` - because of `uncore_imc` type and no entry in custom events,
    it would be counted by **all** Integrated Memory Controller PMUs with config provided from libpfm package.
    (using this function: https://man7.org/linux/man-pages/man3/pfm_get_os_event_encoding.3.html)

- `uncore_imc_0/cas_count_write` - because of `uncore_imc_0` type and entry in custom events it would be counted by `uncore_imc_0` PMU with provided config.

- `uncore_imc_1/cas_count_all` - because of entry in custom events with type field, event would be counted by PMU with **19** type and provided config.

#### Configuring perf events by name

It is possible to configure perf events by names using events supported in [libpfm4](http://perfmon2.sourceforge.net/), for detailed information please see [libpfm4 documentation](http://perfmon2.sourceforge.net/docs_v4.html).

Discovery of perf events supported on platform can be made using python script - [pmu.py](https://sourceforge.net/p/perfmon2/libpfm4/ci/master/tree/python/src/pmu.py) provided with libpfm4, please see [script reqirements](https://sourceforge.net/p/perfmon2/libpfm4/ci/master/tree/python/README).

##### Example configuration of perf events using event names supported in libpfm4

Example output of `pmu.py`:
```
$ python pmu.py
INSTRUCTIONS 1
		 u 0
		 k 1
		 period 3
		 freq 4
		 precise 5
		 excl 6
		 mg 7
		 mh 8
		 cpu 9
		 pinned 10
INSTRUCTION_RETIRED 192
		 e 2
		 i 3
		 c 4
		 t 5
		 intx 7
		 intxcp 8
		 u 0
		 k 1
		 period 3
		 freq 4
		 excl 6
		 mg 7
		 mh 8
		 cpu 9
		 pinned 10
UNC_M_CAS_COUNT 4
		 RD 3
		 WR 12
		 e 0
		 i 1
		 t 2
		 period 3
		 freq 4
		 excl 6
		 cpu 9
		 pinned 10
```
and perf events configuration for listed events:
```json
{
  "core": {
    "events": [
      "instructions",
      "instruction_retired"
    ]
  },
  "uncore": {
    "events": [
      "uncore_imc/unc_m_cas_count:rd",
      "uncore_imc/unc_m_cas_count:wr"
    ]
  }
}
```

Notice: PMU_PREFIX is provided in the same way as for configuration with config values.

#### Grouping

```json
{
  "core": {
    "events": [
      ["instructions", "instruction_retired"]
    ]
  },
  "uncore": {
    "events": [
      ["uncore_imc_0/unc_m_cas_count:rd", "uncore_imc_0/unc_m_cas_count:wr"],
      ["uncore_imc_1/unc_m_cas_count:rd", "uncore_imc_1/unc_m_cas_count:wr"]
    ]
  }
}
```


### Further reading

* [perf Examples](http://www.brendangregg.com/perf.html) on Brendan Gregg's blog
* [Kernel Perf Wiki](https://perf.wiki.kernel.org/index.php/Main_Page)
* `man perf_event_open`
* [perf subsystem](https://github.com/torvalds/linux/tree/v5.6/kernel/events) in Linux kernel
* [Uncore Performance Monitoring Reference Manuals](https://software.intel.com/content/www/us/en/develop/articles/intel-sdm.html#uncore)

See example configuration below:
```json
{
  "core": {
    "events": [
      "instructions",
      "instructions_retired"
    ],
    "custom_events": [
      {
        "type": 4,
        "config": [
          "0x5300c0"
        ],
        "name": "instructions_retired"
      }
    ]
  },
  "uncore": {
    "events": [
      "uncore_imc/cas_count_read"
    ],
    "custom_events": [
      {
        "config": [
          "0xc04"
        ],
        "name": "uncore_imc/cas_count_read"
      }
    ]
  }
}
```

In the example above:
* `instructions` will be measured as a non-grouped event and is specified using human friendly interface that can be 
obtained by calling `perf list`. You can use any name that appears in the output of `perf list` command. This is 
interface that majority of users will rely on.
* `instructions_retired` will be measured as non-grouped event and is specified using an advanced API that allows
to specify any perf event available (some of them are not named and can't be specified with plain string). Event name 
should be a human readable string that will become a metric name.
* `cas_count_read` will be measured as uncore non-grouped event on all Integrated Memory Controllers Performance Monitoring Units because of unset `type` field and
`uncore_imc` prefix.


## Storage driver specific instructions:

* [InfluxDB instructions](storage/influxdb.md).
* [ElasticSearch instructions](storage/elasticsearch.md).
* [Kafka instructions](storage/kafka.md).
* [Prometheus instructions](storage/prometheus.md).
