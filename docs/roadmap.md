# cAdvisor project roadmap

Last updated: Jan 2026.
Status: proposed by SIG Node.

Next steps:

- Otel community agreement
- detailed plans

## Motivation

The motivation for this document is a set of new requests for cAdvisor standalone mode. These requests is a reminder that we need to define a cAdvisor roadmap in light of a current developments in K8s project and a modern landscape of tools and projects.

## Background

CAdvisor consists of two parts that are interleaved and interconnected:

- a library linked into kubelet used to provide information about resource usage that K8s project depends on.
- standalone binary for users to monitor containerized workloads. This includes, but not limited to kubernetes. Outside of kubernetes it is docker containers, Mesos, etc. and provide support for dedicated hardware such as Intel PMU perf metrics.

The project was originated when the industry looked very differently and not well aligned with modern landscape of tools and projects.

Also, since cAdvisor is a Google project, rather than a kubernetes project, results in a few issues:

- Google outsized ownership and responsibility for this project with limited OSS governance model.
- K8s contributors that currently maintains it has little vested interest in owning the standalone mode scenarios.
- Every release of kubernetes requires cAdvisor to be vendored into k8s tree, which can cause significant “dependency hell” and limits what standalone mode cAdvisor can do.

There is ongoing work to deprecate the cAdvisor as a vendored library for k8s. The work consist of multiple stages. First stage is tracked as part of this enhancement:  [cAdvisor-less, CRI-full Container and Pod Stats](https://github.com/kubernetes/enhancements/issues/2371). There will be more work needed after the KEP above is merged to transition machine-wide metrics to containerd.

Once the transition of metrics to container runtime complete, the cAdvisor project will not be in scope of k8s contributors and will be more aligned with the OpenTelemetry project.

Also unless SIG instrumentation will take it as their scope, we will deprecate the cAdvisor endpoint and not accept requests like this: [Configurable cAdvisor Metrics Collection](https://github.com/kubernetes/enhancements/pull/5776) as those are also more aligned with the telemetry scenarios rather than orchestration.

## cAdvisor roadmap

### K8s side

#### 2026

- finish transition of Pod merics to K8s: [cAdvisor-less, CRI-full Container and Pod Stats](https://github.com/kubernetes/enhancements/issues/2371)
- start transition of machine-level metrics to Container Runtime (KEP: TBD).  
- Decide on the future of `cadvisor` endpoint - likely deprecation of the `cadvisor` endpoint of kubelet

#### 2027

- Stop vendoring cAdvisor to K8s.

#### 2027+ (maybe way passed it)

- Remove the `cadvisor` endpoint of kubelet

### cAdvisor standalone

The proposal is to move cAdvisor standalone scenarios to Otel collector. New [receivers](https://opentelemetry.io/docs/collector/components/receiver/) will collect similar information to what cAdvisor collects today and Otel collector can be configured to export as Prometheus endpoint or [any other supported exporter](https://opentelemetry.io/docs/collector/components/exporter/). So there will be a transition path from cAdvisor standalone to Otel collector.

The same time cAdvisor project will be placed in maintenance mode and eventually closed.

If there will be interest from any project or 3rd party to pick up the project and transfer it to CNCF or other project - there will be an open discussion on this.

## Notes on transition to Otel Collector

This is not a detailed spec on transition to Otel Collector. The intent of this section is to highlight the tension points that needs to be clarified as the transition specs are written.

### Distribution model

cAdvisor is a specialized software for containers metrics collection. It is easier to configure and likely more lightweight. Otel collector will need to be compiled with the right set of receivers and exporters and configured appropriately.

The benefit of using Otel collector for these scenarios is it's flexibility in collecting telemetry for more scenarios and a well-knowns config as oppose to a custom cAdvisor configuration.

The believe of this roadmap is that benefits of cAdvisor's distribution model will not be significant enough to prefer it over the Otel collector.

### Supported metrics

The proposal is to transition all metrics to Otel collector to make sure a smooth transition for time-proof set of metrics. However the detailed plan is needed to make a final decision.

The design may split metrics into multiple receivers if this will be more convenient.

### Supported endpoints

Beyond prometheus metrics, cAdvisor supports a few more endpoint. The list is here: https://github.com/google/cadvisor/blob/master/docs/api.md

### prometheus

Prometheus endpoints may expose metrics in a format different as Otel collector prefer dots in names to underscores. Also other renames may potentially be needed as metric names will be aligned with Otel Semantic Convention.

Otel collector will expose the single prometheus endpoint with all metrics for all containers and will not support [container-specific endpoints](https://github.com/google/cadvisor/blob/master/docs/application_metrics.md#api-access-to-application-specific-metrics).

### events

The proposal is to collect events as Otel events.

There is no analogous of events endpoint in Otel collector. So transition to Otel events will require to change design of consumers from pull model to push model for events.

#### docker

Not supported. No alternatives offered.

#### subcontainers

Not supported. No alternatives offered.

#### containers

Not supported. No alternatives offered.

#### machine

Not supported, use [Node Feature Discovery](https://github.com/kubernetes-sigs/node-feature-discovery).

### Supported runtimes

The proposal is to transition container-based metrics collection to Otel collector so all advanced metrics and non-Kubernetes environments monitoring is supported by Otel collector. This way Otel collector will support the same set of environments and runtimes as cAdvisor does today. This may be done by copying code from cAdvisor.

However, as a more lightweight implementation, Otel collector may also implement receivers that are based on CRI-exposed metrics based on KEP [cAdvisor-less, CRI-full Container and Pod Stats](https://github.com/kubernetes/enhancements/issues/2371). These receivers will be less flexible and limited to Kubernetes scenarios. However these receivers will be more reliable.

The proposal is to design for the full set of scenarios, but not limiting to it and allow for CRI-based metrics receivers.

### Supported OSes

cAdvisor does NOT support Windows containers monitoring while Otel collector does. There is a potential for Otel collector to implement this scenario, but it will be outside of a scope of this roadmap document.
