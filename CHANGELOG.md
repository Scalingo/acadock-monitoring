# CHANGELOG

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
