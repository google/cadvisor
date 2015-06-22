# cAdvisor


*French version is available at Docs/FR/README.md*

cAdvisor (Container Advisor) provides container users an understanding of the resource usage and performance characteristics of their running containers. It is a running daemon that collects, aggregates, processes, and exports information about running containers. Specifically, for each container it keeps resource isolation parameters, historical resource usage, histograms of complete historical resource usage and network statistics. This data is exported by container and machine-wide.

cAdvisor has native support for [Docker](https://github.com/docker/docker) containers and should support just about any other container type out of the box. We strive for support accross the board so feel free to open an issue if that is not the case. cAdvisor's container abstraction is based on [lmctfy](https://github.com/google/lmctfy)'s so containers are inherently nested hierarchically.

![cAdvisor](logo.png "cAdvisor")

#### Quick Start: Running cAdvisor in a Docker Container

To quickly tryout cAdvisor on your machine with Docker, we have a Docker image that includes everything you need to get started. You can run a single cAdvisor to monitor the whole machine. Simply run:

```
sudo docker run \
  --volume=/:/rootfs:ro \
  --volume=/var/run:/var/run:rw \
  --volume=/sys:/sys:ro \
  --volume=/var/lib/docker/:/var/lib/docker:ro \
  --publish=8080:8080 \
  --detach=true \
  --name=cadvisor \
  google/cadvisor:latest
```

cAdvisor is now running (in the background) on `http://localhost:8080`. The setup includes directories with Docker state cAdvisor needs to observe.

**Note**: If you're running on CentOS, Fedora, RHEL, or are using LXC take a look at our [running instructions](docs/en/running.md).

We have detailed [instructions](docs/en/running.md#standalone) on running cAdvisor standalone outside of Docker. cAdvisor [running options](docs//en/runtime_options.md) may also be interesting for advanced usecases. If you want to build your own cAdvisor Docker image see our [deployment](docs/en/deploy.md) page.

## Building and Testing

See the more detailed instructions in the [build page](docs/build.md). This includes instructions for building and deploying the cAdvisor Docker image.

## InfluxDB and Cluster Monitoring

cAdvisor supports exporting stats to [InfluxDB](http://influxdb.com). See the [documentation](docs/influxdb.md) for more information and examples.

cAdvisor also exposes container stats as [Prometheus](http://prometheus.io) metrics. See the [documentation](docs/prometheus.md) for more information.

[Heapster](https://github.com/GoogleCloudPlatform/heapster) enables cluster wide monitoring of containers using cAdvisor.

## Web UI

cAdvisor exposes a web UI at its port:

`http://<hostname>:<port>/`

See the [documentation](docs/en/web.md) for more details.

## Remote REST API & Clients

cAdvisor exposes its raw and processed stats via a versioned remote REST API. See the API's [documentation](docs/en/api.md) for more information.

There is also an official Go client implementation in the [client](client/) directory. See the [documentation](docs/en/clients.md) for more information.

## Roadmap

cAdvisor aims to improve the resource usage and performance characteristics of running containers. Today, we gather and expose this information to users. In our roadmap:
- Advise on the performance of a container (e.g.: when it is being negatively affected by another, when it is not receiving the resources it requires, etc)
- Auto-tune the performance of the container based on previous advise.
- Provide usage prediction to cluster schedulers and orchestration layers.

## Community

Contributions, questions, and comments are all welcomed and encouraged! cAdvisor developers hang out in [#google-containers](http://webchat.freenode.net/?channels=google-containers) room on [freenode.net](http://freenode.net).  We also have the [google-containers Google Groups mailing list](https://groups.google.com/forum/#!forum/google-containers).
