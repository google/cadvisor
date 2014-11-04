# Running cAdvisor

## With Docker

We have a Docker image that includes everything you need to get started. Simply run:

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

cAdvisor is now running (in the background) on `http://localhost:8080/`. The setup includes directories with Docker state cAdvisor needs to observe.

### CentOS and RHEL

On CentOS and RHEL the cgroup hierarchies are mounted in `/cgroup` so run cAdvisor with an additional Docker option of `--volume=/cgroup:/cgroup \`.

### LXC Docker exec driver

If you are using Docker with the LXC exec driver, then you need to manually specify all cgroup mounts by adding the:

```
  --volume=/cgroup/cpu:/cgroup/cpu \
  --volume=/cgroup/cpuacct:/cgroup/cpuacct \
  --volume=/cgroup/cpuset:/cgroup/cpuset \
  --volume=/cgroup/memory:/cgroup/memory \
  --volume=/cgroup/blkio:/cgroup/blkio \
```

## Standalone

cAdvisor is a static Go binary with no external dependencies. To run it standalone all you should need to do is run it! Note that some data sources may require root priviledges. cAdvisor will gracefully degrade its features to those it can expose with the access given.

```
$ sudo cadvisor
```

cAdvisor is now running (in the foreground) on `http://localhost:8080/`.

## I need help!

We aim to have cAdvisor run everywhere! If you run into issues getting it running, feel free to file an issue. We are very responsive in supporting our users and update our documentation with new setups.
