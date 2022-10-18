# Acadock Monitoring - Docker container monitoring v1.1.0

This webservice provides live data on Docker containers. It takes
data from the Linux kernel control groups and from the namespace of
the container and expose them through a HTTP API.

> The solution is still a work in progress.

## Configuration

From environment

* `PORT`: port to bind (4244 by default)
* `DOCKER_URL`: docker endpoint (http://127.0.0.1:4243 by default)
* `REFRESH_TIME`: number of second between CPU/net refresh (1 by default)
* `PROC_DIR`: mountpoint for procfs (default to /proc)
* `CGROUP_DIR`: mountpoint of cgroups (default to /sys/fs/cgroup)
* `CGROUP_SOURCE`: "docker" or "systemd" (docker by default)
  docker:  /sys/fs/cgroup/:cgroup/memory/docker
  systemd: /sys/fs/cgroup/:cgroup/memory/system.slice/docker-#{id}.slice
* `DEBUG`: output of debugging information (default "false", switch to "true" to enable)

## Docker

Run from docker:

```
docker run -v /sys/fs/cgroup:/host/cgroup:ro         -e CGROUP_DIR=/host/cgroup \
           -v /proc:/host/proc:ro                    -e PROC_DIR=/host/proc \
           -v /var/run/docker.sock:/host/docker.sock -e DOCKER_URL=unix:///host/docker.sock \
           -p 4244:4244 --privileged --pid=host --network=host \
           -d scalingo/acadock-monitoring
```

`--pid=host`: The daemon has to find the real /proc/#{pid}/ns directory to enter a namespace
`--network=host`: Acadock should in the host namespace to access other containers network namespaces (for network metrics)
`--privileged`: Acadock has to enter the other containers namespaces

## API

* Memory consumption

    `GET /containers/:id/mem`

    Return 200 OK
    Content-Type: application/json

```json
{
  "mem_usage": 123,
  "mem_limit": 5000,
  "max_mem_usage": 500,
  "max_swap_mem_usage": 200,
  "swap_mem_usage": 145,
  "swap_mem_limit": 1000
}
```

* CPU usage (percentage)

    Return 200 OK
    Content-Type: text/plain
    `GET /containers/:id/cpu`

* Network usage (bytes and percentage)

    Return 200 OK
    Content-Type: application/json
    `GET /containers/:id/net`

* Mem+CPU+Network for a container

    Return 200 OK
    Content-Type: application/json
    `GET /containers/:id/usage`

* Mem+CPU+Network for **all** containers

    Return 200 OK
    Content-Type: application/json
    `GET /containers/usage`

## Release a New Version

Bump new version number in:

- `CHANGELOG.md`
- `README.md`

Commit, tag and create a new release:

```sh
git add CHANGELOG.md README.md
git commit -m "Bump v1.1.0"
git tag v1.1.0
git push origin master v1.1.0
hub release create v1.1.0
```

The project is using [GoReleaser](https://goreleaser.com) to build its archives.

To build locally the archives, you can use the following command, it will
automatically use the version of the last tag created.

```
goreleaser release --skip-publish --skip-announce --skip-sign --rm-dist
```

Add the builds to the newly created release.

Last, update the default version installed in the [cookbook](https://github.com/Scalingo/cookbook-acadock).
