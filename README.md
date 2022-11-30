# go-container-hud

A simple UI to show running docker containers with metrics/history for cpu/mem.

Using Docker binding [docker](https://pkg.go.dev/github.com/docker/docker/client)
and Dear ImGui binding [giu](https://pkg.go.dev/github.com/AllenDang/giu)
to create a simple UI
- showing all running containers
  - sort by creation time or name
- showing health status of containers, if available
  - unknown <img src="./heart-unknown.png" width="16" height="16"/>
  - unhealthy <img src="./heart-unhealthy.png" width="16" height="16"/>
  - healthy <img src="./heart-healthy.png" width="16" height="16"/>
- buttons to
  - restart <img src="./restart.png" width="16" height="16"/> container
  - stop <img src="./stop.png" width="16" height="16"/> container
- show basic info like container id or image
- cpu bar-graph to show current cpu metric
- hover over cou bar-graph to show history of cpu usage
- memory bar-graph to show current memory metric
- hover over memory bar-graph to show history of memory usage

![screenshot](./screenshot-with-cpu-history.png)
