# cAdvisor Remote REST API

cAdvisor expose ses stats grâce à la  REST API:

`http://<hostname>:<port>/api/<version>/<request>`

La version actuelle est `v1.3`.

Il y a une version beta de la `v2.0` API [available](api_v2.md).

## Version 1.3

La version expose les même choses que la  `v1.2` avec des fonctions additionnels .

### Evenements

Les noms des ressources pour Docker conteneurs sont celles ci:

`/api/v1.3/events/<absolute container name>`


| Paramètres        | Description                                                                                                                | par défaut           |
|-------------------|----------------------------------------------------------------------------------------------------------------------------|-------------------|
| `start_time`      | L'heure de début des événements pour interroger (pour le stream = false)                                                   | Beginning of time |
| `end_time`        | L'heure de fin des événements à interroger (for stream=false)                                                              | Now               |
| `stream`          | Que ce soit pour flux de nouveaux événements à mesure qu'ils surviennent. Si de fausses déclarations événements historiques| false             |
| `subcontainers`   | Whether to also return events for all subcontainers                                                                        | false             |
| `max_events`      | The max number of events to return (for stream=false)                                                                      | 10                |
| `all_events`      | Whether to include all supported event types                                                                               | false             |
| `oom_events`      | Whether to include OOM events                                                                                              | false             |
| `oom_kill_events` | Whether to include OOM kill events                                                                                         | false             |
| `creation_events` | Whether to include container creation events                                                                               | false             |
| `deletion_events` | Whether to include container deletion events                                                                               | false             |

## Version 1.2

This version exposes the same endpoints as `v1.1` with one additional read-only endpoint.

### Docker Conteneurs Informations

The resource name for Docker container information is as follows:

`/api/v1.2/docker/<Docker container name or blank for all Docker containers>`

The Docker name can be either the UUID or the short name of the container. It returns the information of the specified container(s). The information is returned as a list of serialized `ContainerInfo` JSON objects (found in [info/v1/container.go](../info/v1/container.go)).

## Version 1.1

This version exposes the same endpoints as `v1.0` with one additional read-only endpoint.

### Subcontainer Information

The resource name for subcontainer information is as follows:

`/api/v1.1/subcontainers/<absolute container name>`

Where the absolute container name follows the lmctfy naming convention (described bellow). It returns the information of the specified container and all subcontainers (recursively). The information is returned as a list of serialized `ContainerInfo` JSON objects (found in [info/v1/container.go](../info/v1/container.go)).

## Version 1.0

This version exposes two main endpoints, one for container information and the other for machine information. Both endpoints are read-only in v1.0.

### Conteuner Informations

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

The actual object is the marshalled JSON of the `ContainerInfo` struct found in [info/v1/container.go](../info/v1/container.go)

### Machine Information

The resource name for machine information is as follows:

`/api/vX.Y/machine`

This resource is read-only. The machine information is returned as a JSON object containing:

- Number of schedulable logical CPU cores
- Memory capacity (in bytes)
- Maximum supported CPU frequency (in kHz)
- Available filesystems: major, minor numbers and capacity (in bytes)
- Network devices: mac addresses, MTU, and speed (if available)
- Machine topology: Nodes, cores, threads, per-node memory, and caches

The actual object is the marshalled JSON of the `MachineInfo` struct found in [info/v1/machine.go](../info/v1/machine.go)
