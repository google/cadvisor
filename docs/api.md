# cAdvisor Remote REST API

cAdvisor exposes its raw and processed stats via a versioned remote REST API:

`http://<hostname>:<port>/api/<version>/<request>`

The current version of the API is `v1.2`.

## Version 1.2

This version exposes the same endpoints as `v1.1` with one additional read-only endpoint.

### Docker Container Information

The resource name for Docker container information is as follows:

`/api/v1.2/docker/<Docker container name or blank for all Docker containers>`

The Docker name can be either the UUID or the short name of the container. It returns the information of the specified container(s). The information is returned as a list of serialized `ContainerInfo` JSON objects (found in [info/container.go](info/container.go)).

## Version 1.1

This version exposes the same endpoints as `v1.0` with one additional read-only endpoint.

### Subcontainer Information

The resource name for subcontainer information is as follows:

`/api/v1.1/subcontainers/<absolute container name>`

Where the absolute container name follows the lmctfy naming convention (described bellow). It returns the information of the specified container and all subcontainers (recursively). The information is returned as a list of serialized `ContainerInfo` JSON objects (found in [info/container.go](info/container.go)).

## Version 1.0

This version exposes two main endpoints, one for container information and the other for machine information. Both endpoints are read-only in v1.0.

### Container Information

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

### Machine Information

The resource name for machine information is as follows:

`/api/v1.0/machine`

This resource is read-only. The machine information is returned as a JSON object containing:

- Number of schedulable logical CPU cores
- Memory capacity (in bytes)

The actual object is the marshalled JSON of the `MachineInfo` struct found in [info/machine.go](info/machine.go)
