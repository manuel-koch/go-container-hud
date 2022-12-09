package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.design/x/clipboard"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// these vars will be set on build time
	versionTag  string
	versionSha1 string
	buildDate   string
)

var (
	app                *App = nil
	containerInfo           = make(map[string]*ContainerInfo, 0)
	containerInfoMutex      = sync.RWMutex{}
)

func getDockerStats(ctx context.Context) chan bool {
	done := make(chan bool, 1)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("Failed to get docker client: %v\n", err)
		defer close(done)
	}

	// handle container info send thru channel to start following container stats
	newContainerIds := make(chan string, 1)
	go func() {
		for id := range newContainerIds {
			// local copy of newContainer struct
			if _, ok := containerInfo[id]; !ok {
				containerInfoMutex.Lock()
				containerInfo[id] = NewContainerInfo(id)
				info := containerInfo[id]
				if inspect, err := cli.ContainerInspect(ctx, id); err == nil {
					compose_project_dir := inspect.Config.Labels["com.docker.compose.project.working_dir"]
					container_number := 1
					if i, err := strconv.Atoi(inspect.Config.Labels["com.docker.compose.container-number"]); err == nil {
						container_number = i
					}
					if created, err := time.Parse("2006-01-02T15:04:05.000000000Z", inspect.ContainerJSONBase.State.StartedAt); err == nil {
						info.Data.Created = created.Unix()
					}
					info.Data.State = ContainerRunning
					info.Data.Name = strings.TrimLeft(inspect.Name, "/")
					info.Data.Image = inspect.Image
					info.Data.DockerComposeProject = inspect.Config.Labels["com.docker.compose.project"]
					info.Data.DockerComposeProjectDir = compose_project_dir
					info.Data.DockerComposeService = inspect.Config.Labels["com.docker.compose.service"]
					info.Data.DockerComposeContainerNumber = container_number
					info.Data.SetAlternativeName()
				} else {
					fmt.Printf("Failed to inspect container %s: %v", info.Data.ID, err)
				}
				statsCtx, statsCancel := context.WithCancel(context.Background())
				containerInfo[id].OnStopped = func() {
					statsCancel()
					containerInfoMutex.Lock()
					defer containerInfoMutex.Unlock()
					delete(containerInfo, info.Data.ID)
				}
				containerInfo[id].Stop = func() {
					info.mutex.Lock()
					if info.Data.State == ContainerRunning {
						info.Data.State = ContainerStopping
						info.mutex.Unlock()
						fmt.Printf("Stopping container %s (%s)...\n", info.Data.AlternativeName, info.Data.ID)
						err := cli.ContainerStop(ctx, info.Data.ID, nil)
						if err != nil {
							fmt.Printf("Failed to stop container %s (%s): %v", info.Data.AlternativeName, info.Data.ID, err)
						}
					} else {
						info.mutex.Unlock()
					}
				}
				containerInfo[id].Restart = func() {
					info.mutex.Lock()
					defer info.mutex.Unlock()
					if info.Data.State == ContainerRunning {
						info.Data.State = ContainerRestarting
						fmt.Printf("Restarting container %s (%s)...\n", info.Data.AlternativeName, info.Data.ID)
						err := cli.ContainerRestart(ctx, info.Data.ID, nil)
						if err != nil {
							fmt.Printf("Failed to restart container %s (%s): %v", info.Data.AlternativeName, info.Data.ID, err)
						}
					}
				}
				fmt.Printf("Following container: %s (%s)\n", info.Data.AlternativeName, info.Data.ID)
				go updateContainerStats(statsCtx, cli, info)
				containerInfoMutex.Unlock()

				go func() {
					for {
						select {
						case <-ctx.Done():
							// outer context closed, stop stats context too
							statsCancel()
						case <-statsCtx.Done():
							// stats context closed, we're done
							return
						case <-time.After(1 * time.Second):
							//
						}
					}
				}()
			}
		}
	}()

	// listen to docker events related to starting & stopping containers
	eventOptions := types.EventsOptions{}
	events, _ := cli.Events(ctx, eventOptions)
	go func() {
		for event := range events {
			fmt.Printf("Container Event: %s %s %s\n", event.Type, event.Status, event.Action)
			if event.Type == "container" {
				if event.Action == "start" {
					fmt.Printf("Container started: %s\n", event.Actor.ID)
					newContainerIds <- event.Actor.ID
				}
				if event.Action == "stop" || event.Action == "destroy" {
					if info, ok := containerInfo[event.Actor.ID]; ok {
						fmt.Printf("Container stopped: %s (%s)\n", info.Data.AlternativeName, event.Actor.ID)
						info.mutex.Lock()
						info.Data.State = ContainerStopped
						info.mutex.Unlock()
						info.OnStopped()
					}
				}
			}
		}
	}()

	// get currently running containers too
	listOptions := types.ContainerListOptions{}
	containers, err := cli.ContainerList(ctx, listOptions)
	if err != nil {
		fmt.Printf("Failed to get running containers: %v\n", err)
		close(done)
		return done
	}
	for i := range containers {
		fmt.Printf("Container is running: %s\n", containers[i].ID)
		newContainerIds <- containers[i].ID
	}

	go func() {
		for {
			select {
			case <-time.After(5 * time.Second):
				// signal done if docker server is not available
				ping, err := cli.Ping(ctx)
				if err != nil || len(ping.APIVersion) == 0 {
					fmt.Printf("Ping docker server failed: %v\n", err)
					close(done)
					return
				}
				fmt.Println("Ping docker server ok")
			case <-ctx.Done():
				close(done)
				return
			}
		}
	}()

	return done
}

func getDockerStatsWithRetry(ctx context.Context) {
	retryAfter := 5 * time.Second
	go func() {
		for {
			fmt.Println("Following docker stats...")
			statsCtx, cancel := context.WithCancel(context.Background())
			done := getDockerStats(statsCtx)
			select {
			case <-done:
				fmt.Printf("Retrying to follow docker stats in %s...\n", retryAfter)
				containerInfoMutex.Lock()
				containerInfo = make(map[string]*ContainerInfo, 0)
				containerInfoMutex.Unlock()
				cancel()
				time.Sleep(retryAfter)
			case <-ctx.Done():
				cancel()
				return
			}
		}
	}()
}

func restartContainer(id string) {
	containerInfoMutex.RLock()
	if info, ok := containerInfo[id]; ok {
		containerInfoMutex.RUnlock()
		info.Restart()
	} else {
		containerInfoMutex.RUnlock()
	}
}

func stopContainer(id string) {
	containerInfoMutex.RLock()
	if info, ok := containerInfo[id]; ok {
		containerInfoMutex.RUnlock()
		info.Stop()
	} else {
		containerInfoMutex.RUnlock()
	}
}

func sendContainerDataToApp() {
	containerInfoMutex.RLock()
	defer containerInfoMutex.RUnlock()

	data := make([]ContainerData, len(containerInfo))
	i := 0
	for _, info := range containerInfo {
		info.mutex.RLock()
		data[i] = info.Data
		info.mutex.RUnlock()
		i++
	}

	app.ContainerData(data)
}

func main() {
	buildInfo := fmt.Sprintf("v%s\nbuilt %s\ncommit sha1 %s", versionTag, buildDate, versionSha1)
	fmt.Println(buildInfo)

	if err := clipboard.Init(); err != nil {
		panic(fmt.Errorf("Unable to use clipboard: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	getDockerStatsWithRetry(ctx)

	app = NewApp()
	app.BuildInfo(buildInfo)
	app.OnStopContainer(stopContainer)
	app.OnRestartContainer(restartContainer)

	go func() {
		for {
			select {
			case <-time.After(1 * time.Second):
				sendContainerDataToApp()
			case <-ctx.Done():
				return
			}
		}
	}()

	app.Run()

	cancel()
}
