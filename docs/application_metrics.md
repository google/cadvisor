# Collecting Application Metrics with cAdvisor

**Note** Application metrics support is in Alpha. We are still making a bunch of interface changes.

## Introduction
In addition to usage metrics, cAdvisor can also be configured to collect application metrics. A container can expose application metrics through multiple ways - on a status page, through structured info like prometheus, or have a separate API for fetching stats. cAdvisor provides a generic way to collect these metrics. Additional templates are provided to automate some well-known collection profiles.

## Specifying application metrics

Application metrics specification consists of two steps:
* Creating a configuration
* Passing the configuration location to cadvisor

## Creating a configuration
An application metric configuration tells cAdvisor where to look for application metrics and specifies other parameters about how to export the metrics from cAdvisor to UI and backends. The metric config includes:
* Endpoint (Location to collect metrics from)
* Name of metric
* Type (Counter, Gauge, ...)
* Data Type (int, float)
* Units (kbps, seconds, count)
* Polling Frequency
* Regexps (Regular expressions to specify which metrics to collect and how to parse them)

### Endpoint configuration
The endpoint can be specified in two ways.

If you know the full URL before hand, you can directly specify it as a string:
```
"endpoint" : "http://myHost.example.com:8000/metrics"
```

When dealing with containers you may not know the ip address of the container before hand, and in this case we cannot directly enter the URL.

For these situations, you can specify the `protocol`, `port` and `path` values and cAdvisor can generate the full URL for you.
```
"endpoint" : {
  "protocol" : "http",
  "port" : 8000,
  "path" : "/metrics"
}
```  
From this information, cAdvisor will generate a url of type `http://${container_ip_address}:8000/metrics` where the `${container_ip_address}` is the ip address of the container.

### Sample Generic Metric Configuration

Here is an example of a very generic metric collector that assumes no structured information:

```
{
  "endpoint" : "http://localhost:8000/nginx_status",
  "metrics_config" : [
    {
      "name" : "activeConnections",
      "metric_type" : "gauge",
      "units" : "number of active connections",
      "data_type" : "int",
      "polling_frequency" : "10s",
      "regex" : "Active connections: ([0-9]+)"
    },
    {
      "name" : "reading",
      "metric_type" : "gauge",
      "units" : "number of reading connections",
      "data_type" : "int",
      "polling_frequency" : "10s",
      "regex" : "Reading: ([0-9]+) .*"
    }
  ]
} 
```
This endpoint exposes metrics in a non structured manner and the cAdvisor collector uses regular expressions to determine where the metrics are to be collected.

#### Sample Prometheus Configuration

Endpoint which expose metrics in a Prometheus format do so in a more structured format and cAdvisor can gather information about the metrics directly from the endpoint.

Here is a sample prometheus config that collects all metrics from an endpoint:

```
{
  "endpoint" : "http://localhost:9100/metrics"
}
```

If you only want to collect a subset of metrics from an endpoint, you can specify which metrics to collect via the `metrics_config` parameter.

This sample config only collects only the `scheduler_binding_latency`, `scheduler_e2e_scheduling_latency` and `scheduling_algorithm_latency` metrics from the endpoint:

```
{
  "endpoint" : "http://localhost:8000/metrics",
  "metrics_config" : [
    "scheduler_binding_latency",
    "scheduler_e2e_scheduling_latency",
    "scheduling_algorithm_latency"
  ]
}
```
#### Sample Jolokia Configuration

For Java applications, Jolokia can be used to expose metrics stored in mbeans.

The following configuration will gather the currently used heap memory by the Java application:

```
{
  "endpoint" : {
    "protocol" : "https",
    "path" : "/jolokia",
    "port" : 8778
  },
  "polling_frequency" : "10s",
  "metrics_config" : [
    { "name": "JVMHeap",
      "mbean": {
        "name" : "java.lang:type=Memory",
        "attribute": "HeapMemoryUsage",
        "path": "used"
      },
      "metric_type": "gauge",
      "units": "bytes",
      "data_type": "int"
    }
  ]
}
```

## Passing the configuration to cAdvisor

cAdvisor can discover any configurations for a container using Docker container labels. Any label starting with ```io.cadvisor.metric``` is parsed as a cadvisor application-metric label.
cAdvisor uses the value as an indicator of where the configuration can be found.  Labels of the form ```io.cadvisor.metric.prometheus-xyz``` indicate that the configuration points to a
Prometheus metrics endpoint. Labels of the form ```io.cadvisor.metric.jolokia-xyz``` indicate that the configuration points to a Jolokia endpoint.

The configuration file can either be part of the container image or can be added on at runtime with a volume. This makes sure that there is no connection between the host where the container is running and the application metrics configuration. A container is self-contained for its metric information.

So a sample configuration for redis would look like:

Dockerfile (or runtime):
```
 FROM redis
 ADD ADD redis_config.json /var/cadvisor/redis_config.json
 LABEL io.cadvisor.metric.redis="/var/cadvisor/redis_config.json"
```

cAdvisor will then reach into the container image at runtime, process the config, and start collecting and exposing application metrics.

Note that cAdvisor specifically looks at the container labels to extract this information.  In Docker 1.8, containers don't inherit labels
from their images, and thus you must specify the label at runtime.

## API access to application-specific metrics

A new endpoint is added for collecting application-specific metrics for a particular container:

```
http://localhost:8080/api/v2.0/appmetrics/containerName
```

The set of application-metrics being collected can be discovered from the container spec:

```
http://localhost:8080/api/v2.0/spec/containerName
```

Regular stats API also has application-metrics appended to it:

```
http://localhost:8080/api/v2.0/stats/containerName
```

## UI changes
Application-metrics show up on the container page after the resource metrics.

## Ongoing work

### Templates
Next step for application-metrics is to add templates for well-known containers that have stable stats API. These would be specified by a new label ```io.cadvisor.metric.type```. If the label value is a known type, cAdvisor would start collecting stats automatically without needing any further config. Config can still be used to override any specific parameters - like set of metrics to collect. 

### UI enhancements
There are a bunch of UI enhancements under way:
* Better handling/display of metrics - eg. allowing overlaying metrics on the same graphs, handling metric types like percentiles.
* Moving application metrics to separate tab.
* Adding control to show only selected metrics on UI while still exporting everything through the API.
