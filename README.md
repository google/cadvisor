# cAdvisor

cAdvisor (Container Advisor) provides container users an understanding of the resource usage and performance characteristics of their running containers. It is a running daemon that collects, aggregates, processes, and exports information about running containers. Specifically, for each container it keeps resource isolation parameters, historical resource usage, histograms of complete historical resource usage and network statistics. This data is exported by container and machine-wide.

cAdvisor currently supports lmctfy containers as well as Docker containers (those that use the default libcontainer execdriver). Other container backends can also be added. cAdvisor's container abstraction is based on lmctfy's so containers are inherently nested hierarchically.

![cAdvisor](logo.png "cAdvisor")

#### Quick Start: Running cAdvisor in a Docker Container

To quickly tryout cAdvisor on your machine with Docker (version 0.11 or above), we have a Docker image that includes everything you need to get started. Simply run:

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

**Note**: On CentOS and RHEL 6, run cAdvisor with an additional option ```--volume=/cgroup:/cgroup \```

If you want to build your own cAdvisor Docker image, take a look at `deploy/Dockerfile` and `deploy/build.sh`.

#### Using [InfluxDB](http://influxdb.com) as backend storage

cAdvisor now also supports [InfluxDB](http://influxdb.com) to store stats. To use InfluxDB, you need to pass some additional flags to cAdvisor:

**Required**
```
 # storage driver to use. Options are: memory (default) and influxdb
 -storage_driver=influxdb
 # Required to make glog work
 -log_dir=/
```
**Optional**
```
 # The *ip:port* of the database. Default is 'localhost:8086'
 -storage_driver_host=ip:port
 # database name. Uses db 'cadvisor' by default
 -storage_driver_name
 # database username. Default is 'root'
 -storage_driver_user
 # database password. Default is 'root'
 -storage_driver_password
 # Use secure connection with database. False by default
 -storage_driver_secure
```

## Cluster monitoring using cAdvisor

[Heapster](https://github.com/GoogleCloudPlatform/heapster) enables cluster wide monitoring of containers using cAdvisor.

## Web UI

cAdvisor exposes a web UI at its port:

`http://<hostname>:<port>/`

## Remote REST API

cAdvisor exposes its raw and processed stats via a versioned remote REST API:

`http://<hostname>:<port>/api/<version>/<request>`

The current version of the API is `v1.2`.

### Version 1.2

This version exposes the same endpoints as `v1.1` with one additional read-only endpoint.

#### Docker Container Information

The resource name for Docker container information is as follows:

`/api/v1.1/docker/<Docker container name or blank for all Docker containers>`

The Docker name can be either the UUID or the short name of the container. It returns the information of the specified container(s). The information is returned as a list of serialized `ContainerInfo` JSON objects (found in [info/container.go](info/container.go)).

### Version 1.1

This version exposes the same endpoints as `v1.0` with one additional read-only endpoint.

#### Subcontainer Information

The resource name for subcontainer information is as follows:

`/api/v1.1/subcontainers/<absolute container name>`

Where the absolute container name follows the lmctfy naming convention (described bellow). It returns the information of the specified container and all subcontainers (recursively). The information is returned as a list of serialized `ContainerInfo` JSON objects (found in [info/container.go](info/container.go)).

### Version 1.0

This version exposes two main endpoints, one for container information and the other for machine information. Both endpoints are read-only in v1.0.

#### Container Information

The resource name for container information is as follows:

`/api/v1.0/containers/<absolute container name>`

Where the absolute container name follows the lmctfy naming convention. For example:

| Container Name       | Resource Name                             |
|----------------------|-------------------------------------------|
| /                    | /api/v1.0/containers/                     |
| /foo                 | /api/v1.0/containers/foo                  |
| /docker/2c4dee605d22 | /api/v1.0/containers/docker/2c4dee605d22  |

Note that the root container (`/`) contains usage for the entire machine. All Docker containers are listed under `/docker`.

The container information is returned as a JSON object containing:

- Absolute container name
- List of subcontainers
- ContainerSpec which describes the resource isolation enabled in the container
- Detailed resource usage statistics of the container for the last `N` seconds (`N` is globally configurable in cAdvisor)
- Histogram of resource usage from the creation of the container

The actual object is the marshalled JSON of the `ContainerInfo` struct found in [info/container.go](info/container.go)

#### Machine Information

The resource name for machine information is as follows:

`/api/v1.0/machine`

This resource is read-only. The machine information is returned as a JSON object containing:

- Number of schedulable logical CPU cores
- Memory capacity (in bytes)

The actual object is the marshalled JSON of the `MachineInfo` struct found in [info/machine.go](info/machine.go)

## REST API Clients

There is an example Go client under [client/](client/) - you can use it on your own Go project by including it like this:

```go
import "github.com/google/cadvisor/client"
client, err = client.NewClient("http://192.168.59.103:8080/")
mInfo, err := client.MachineInfo()
```

## Roadmap

cAdvisor aims to improve the resource usage and performance characteristics of running containers. Today, we gather and expose this information to users. In our roadmap:
- Advise on the performance of a container (e.g.: when it is being negatively affected by another, when it is not receiving the resources it requires, etc)
- Auto-tune the performance of the container based on previous advise.
- Provide usage prediction to cluster schedulers and orchestration layers.

## Community

Contributions, questions, and comments are all welcomed and encouraged! cAdvisor developers hang out in [#google-containers](http://webchat.freenode.net/?channels=google-containers) room on freenode.net.  We also have the [google-containers Google Groups mailing list](https://groups.google.com/forum/#!forum/google-containers).
