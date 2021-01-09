# Running cAdvisor

## With Docker

We have a Docker image that includes everything you need to get started. Simply run:

```
VERSION=v0.35.0 # use the latest release version from https://github.com/google/cadvisor/releases
sudo docker run \
  --volume=/:/rootfs:ro \
  --volume=/var/run:/var/run:rw \
  --volume=/sys:/sys:ro \
  --volume=/var/lib/docker/:/var/lib/docker:ro \
  --publish=8080:8080 \
  --detach=true \
  --name=cadvisor \
  gcr.io/cadvisor/cadvisor:$VERSION
```

cAdvisor is now running (in the background) on `http://localhost:8080/`. The setup includes directories with Docker state cAdvisor needs to observe.

**Note**: 
- If docker daemon is running with [user namespace enabled](https://docs.docker.com/engine/reference/commandline/dockerd/#starting-the-daemon-with-user-namespaces-enabled),
you need to add `--userns=host` option in order for cAdvisor to monitor Docker containers,
otherwise cAdvisor can not connect to docker daemon.
- If cadvisor scrapes `process metrics` by set flag `--disable_metrics`, you need to add `--pid=host` and `--privileged` for `docker run` to get `/proc/pid/fd` path in host.
- If cAdvisor needs to be run in Docker container without `--privileged` option it is possible to add host devices to container using `--dev` and
  specify security options using `--security-opt` with secure computing mode (seccomp).
  For details related to seccomp please [see](https://docs.docker.com/engine/security/seccomp/), the default Docker profile can be found [here](https://github.com/moby/moby/blob/master/profiles/seccomp/default.json).

  For example to run cAdvisor with perf support in Docker container without `--privileged` option it is required to:
  - Set perf_event_paranoid using `sudo sysctl kernel.perf_event_paranoid=-1`, see [documentation](https://www.kernel.org/doc/Documentation/sysctl/kernel.txt)
  - Add "perf_event_open" syscall into syscalls array with the action: "SCMP_ACT_ALLOW" in [default Docker profile](https://github.com/moby/moby/blob/master/profiles/seccomp/default.json)
  - Run Docker container with following options:
  ```
  docker run \
  --volume=/:/rootfs:ro \
  --volume=/var/run:/var/run:ro \
  --volume=/sys:/sys:ro \
  --volume=/var/lib/docker/:/var/lib/docker:ro \
  --volume=/dev/disk/:/dev/disk:ro \
  --volume=$GOPATH/src/github.com/google/cadvisor/perf/testing:/etc/configs/perf \
  --publish=8080:8080 \
  --device=/dev/kmsg \
  --security-opt seccomp=default.json \
  --name=cadvisor \
  gcr.io/cadvisor/cadvisor:<tag> -perf_events_config=/etc/configs/perf/perf.json
  ```

## Latest Canary

The latest cAdvisor canary release is continuously built from HEAD and available
as an Automated Build Docker image:
[google/cadvisor-canary](https://registry.hub.docker.com/u/google/cadvisor-canary/). We do *not* recommend using this image in production due to its large size and volatility.

## With Boot2Docker

After booting up a boot2docker instance, run cAdvisor image with the same docker command mentioned above. cAdvisor can now be accessed at port 8080 of your boot2docker instance. The host IP can be found through DOCKER_HOST environment variable setup by boot2docker:

```
$ echo $DOCKER_HOST
tcp://192.168.59.103:2376
```

In this case, cAdvisor UI should be accessible on `http://192.168.59.103:8080`

## Other Configurations

### CentOS, Fedora, and RHEL

You may need to run the container with `--privileged=true` and `--volume=/cgroup:/cgroup:ro \` in order for cAdvisor to monitor Docker containers.

RHEL and CentOS lock down their containers a bit more. cAdvisor needs access to the Docker daemon through its socket. This requires `--privileged=true` in RHEL and CentOS.

On some versions of RHEL and CentOS the cgroup hierarchies are mounted in `/cgroup` so run cAdvisor with an additional Docker option of `--volume=/cgroup:/cgroup:ro \`.

**Note**: For a RedHat 7 docker host the default run commands from above throw oci errors. Please use the command below if the host is RedHat 7:
```
docker run
--volume=/:/rootfs:ro
--volume=/var/run:/var/run:rw
--volume=/sys/fs/cgroup/cpu,cpuacct:/sys/fs/cgroup/cpuacct,cpu
--volume=/var/lib/docker/:/var/lib/docker:ro
--publish=8080:8080
--detach=true
--name=cadvisor
--privileged=true
google/cadvisor:latest
```


### Debian

By default, Debian disables the memory cgroup which does not allow cAdvisor to gather memory stats. To enable the memory cgroup take a look at [these instructions](https://github.com/google/cadvisor/issues/432).

### LXC Docker exec driver

If you are using Docker with the LXC exec driver, then you need to manually specify all cgroup mounts by adding the:

```
  --volume=/cgroup/cpu:/cgroup/cpu \
  --volume=/cgroup/cpuacct:/cgroup/cpuacct \
  --volume=/cgroup/cpuset:/cgroup/cpuset \
  --volume=/cgroup/memory:/cgroup/memory \
  --volume=/cgroup/blkio:/cgroup/blkio \
```

### Invalid Bindmount `/`

This is a problem seen in older versions of Docker. To fix, start cAdvisor without the `--volume=/:/rootfs:ro` mount. cAdvisor will degrade gracefully by dropping stats that depend on access to the machine root.

## Standalone

cAdvisor is a static Go binary with no external dependencies. To run it standalone all you should need to do is run it! Note that some data sources may require root privileges. cAdvisor will gracefully degrade its features to those it can expose with the access given.

```
$ sudo cadvisor
```

cAdvisor is now running (in the foreground) on `http://localhost:8080/`.

## Runtime Options

cAdvisor has a series of flags that can be used to configure its runtime behavior. More details can be found in runtime [options](runtime_options.md).

## Hardware Accelerator Monitoring

cAdvisor can export some metrics for hardware accelerators attached to containers.
Currently only Nvidia GPUs are supported. There are no machine level metrics.
So, metrics won't show up if no container with accelerators attached is running.
Metrics will only show up if accelerators are explicitly attached to the container, e.g., by passing `--device /dev/nvidia0:/dev/nvidia0` flag to docker.
If nothing is explicitly attached to the container, metrics will NOT show up. This can happen when you access accelerators from privileged containers.

There are two things that cAdvisor needs to show Nvidia GPU metrics:
- access to NVML library (`libnvidia-ml.so.1`).
- access to the GPU devices.

If you are running cAdvisor inside a container, you will need to do the following to give the container access to NVML library:
```
-e LD_LIBRARY_PATH=<path-where-nvml-is-present>
--volume <above-path>:<above-path>
```

If you are running cAdvisor inside a container, you can do one of the following to give it access to the GPU devices:
- Run with `--privileged`
- If you are on docker v17.04.0-ce or above, run with `--device-cgroup-rule 'c 195:* mrw'`
- Run with `--device /dev/nvidiactl:/dev/nvidiactl /dev/nvidia0:/dev/nvidia0 /dev/nvidia1:/dev/nvidia1 <and-so-on-for-all-nvidia-devices>`
