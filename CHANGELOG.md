# Changelog

## To Be Released

* Compatibility with cgroup v2: use github.com/containerd/cgroup to read cgroup data for v1 and v2
* chore(go): use Go 1.24
* build(go.mod): update `github.com/urfave/negroni` from v1 to v3
* chore(go): use Go 1.22.10
* use github.com/Scalingo/go-utils/graceful for graceful upgrades and shutdowns
* use github.com/Scalingo/go-utils/errors/v3 for errors management

## v1.2.1 - 2023-12-27

* chore(deps): various updates

## v1.2.0 - 2022-12-01

* Bump github.com/fsouza/go-dockerclient from 1.8.3 to 1.9.0
* Bump github.com/Scalingo/go-handlers from 1.4.5 to 1.5.0
* Bump github.com/tklauser/go-sysconf from 0.3.10 to 0.3.11
* Wrap all errors of the project
* Replace `errgo` package by `pkg/errors` for errors handling
* Don't return an error when memory can't be fetched for one container,
  just skip the container and return normally metrics at the end

## v1.1.0 - 2022-10-18

* Add Error Middleware to the router
* Bump github.com/Scalingo/go-handlers from 1.4.4 to 1.4.5
* Bump github.com/fsouza/go-dockerclient from 1.8.1 to 1.8.3
* Bump github.com/Scalingo/go-utils/logger from 1.1.1 to 1.2.0
* Bump github.com/Scalingo/go-handlers from 1.4.3 to 1.4.4
* Bump github.com/stretchr/testify from v1.7.1 to v1.8.0
* Bump github.com/fsouza/go-dockerclient from v1.7.11 to v1.8.1
* Bump github.com/Scalingo/go-utils/logger from v1.1.0 to v1.1.1
* Bump github.com/sirupsen/logrus from 1.8.1 to 1.9.0

## v1.0.1

* chore(go): use go 1.17
* Bump various dependencies

## v1.0.0

* Bump Go version to 1.16
* Bump github.com/golang/mock from 1.3.1 to 1.5.0
* Bump github.com/stretchr/testify from 1.5.1 to 1.7.0
* Bump github.com/Scalingo/go-handlers from 1.3.1 to 1.4.0
* Bump github.com/sirupsen/logrus from 1.7.0 to 1.8.1
* Bump github.com/tklauser/go-sysconf from v0.0.0-20200513113950-67a71062da8a to 0.3.9

## v0.6.1

* Bump dependencies, stop using github.com/Scalingo/go-utils globally and uses submodules

## v0.6.0

* Add Host resources monitoring
* Add authentication

## v0.5.1

* Fix CPU value reading

## v0.5.0

* Fix int overflow when parsing cpuacct cgroup file
* Ignore first values of net/cpu monitoring to avoid extreme values (return 0 instead)
* Migration from godep to dep

## v0.4.1

* Improve logging using logrus, less spammy
* Replace martini by gorilla/mux and negroni

## v0.4.0

* Net monitoring improvement
  -> Just setns/fork to find the host network interface ID, then directly
     read `veth` value from the host without having to enter the namespace
     again

## v0.3.6

* Fix race condition

## v0.3.5

* Fix memory monitoring: negative swap values were possible before, not anymore

## v0.3.4

* Fix logic in net monitoring

## v0.3.3

* Let user disable Net monitoring

## v0.3.2

* BUGFIX fd leak when reading cgroup file
* NEW add labels to /containers/usage to get metadata of containers

## v0.3.1

* NEW /containers/usage to get cpu + mem for all containers

## v0.3.0

* /containers/:id/cpu returns an object with `.usage_in_percents`
* /containers/:id/memory drops `.swap_memory_usage`, `.swap_memory_limit` and `.max_swap_memory`
* /containers/:id/memory get `.swap_usage`, `.swap_limit` and `.max_swap_usage`
* NEW /containers/:id/usage to get cpu + mem and optionally net usage

## v0.2.4

* Remove dependency to app from API client (client package)

## v0.2.3

* Fix nil exception when looking for memory data

## v0.2.2

* Update client package to fit new API

## v0.2.1

* Change runner PATH model
* Rename net runner to acadock-monitoring-ns-netstat

## v0.2.0

* Can run in a container
* Stats of net interface of containers
* More memory stats

## v0.1.2

* CPU monitoring "not ready" -> -1

## v0.1.1

* Parse memory on 64bits

## v0.1.0

* Metrics with Docker - Systemd
* Metrics with Docker - Libcontainer
