package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"math"
	"strings"
	"sync"
	"time"
)

type ContainerInfo struct {
	mutex sync.RWMutex
	Data  ContainerData

	OnUpdated func()
	OnStopped func()

	Stop    func()
	Restart func()
}

type ContainerData struct {
	ID                           string
	State                        ContainerState
	Created                      int64
	Name                         string
	AlternativeName              string
	Image                        string
	DockerComposeProject         string
	DockerComposeProjectDir      string
	DockerComposeService         string
	DockerComposeContainerNumber int
	EnvVars                      map[string]string

	LastUpdated       int64
	CpuPercent        float64
	CpuPercentHistory History
	Memory            uint64
	MemoryLimit       uint64
	MemoryPercent     float64
	MemoryHistory     History
	NetworkTx         uint64
	NetworkTxHistory  History
	NetworkRx         uint64
	NetworkRxHistory  History
	BlockRead         uint64
	BlockWrite        uint64
	PIDs              uint64
	HealthUpdated     int64
	HealthStatus      HealthState
}

type ContainerState int

const (
	ContainerUnknownState ContainerState = iota
	ContainerRunning      ContainerState = iota
	ContainerRestarting   ContainerState = iota
	ContainerStopping     ContainerState = iota
	ContainerStopped      ContainerState = iota
)

type HealthState int

const (
	UnknownHealth HealthState = iota
	Healthy       HealthState = iota
	Unhealthy     HealthState = iota
)

func NewContainerData(id string) ContainerData {
	return ContainerData{
		ID:                id,
		State:             ContainerUnknownState,
		CpuPercentHistory: NewHistory(),
		MemoryHistory:     NewHistory(),
		EnvVars:           make(map[string]string, 0),
	}
}

func (d *ContainerData) SetAlternativeName() {
	if len(d.DockerComposeService) > 0 && d.DockerComposeContainerNumber > 0 {
		d.AlternativeName = fmt.Sprintf("%s-%d", d.DockerComposeService, d.DockerComposeContainerNumber)
	} else if len(d.DockerComposeService) > 0 {
		d.AlternativeName = d.DockerComposeService
	} else if len(d.Name) > 0 {
		d.AlternativeName = d.Name
	} else {
		d.AlternativeName = fmt.Sprintf("%8.8s", d.ID)
	}
}

func NewContainerInfo(id string) *ContainerInfo {
	return &ContainerInfo{Data: NewContainerData(id)}
}

func (c *ContainerInfo) Updated() {
	if c.OnUpdated != nil {
		c.OnUpdated()
	}
}

// functionality copied from
// https://github.com/moby/moby/blob/eb131c5383db8cac633919f82abad86c99bffbe5/cli/command/container/stats_helpers.go

func updateContainerStats(ctx context.Context, cli *client.Client, container *ContainerInfo) {
	ctx_ := ctx
	response, err := cli.ContainerStats(ctx, container.Data.ID, true)
	if err != nil {
		panic(err)
	}

	var (
		previousCPU    uint64
		previousSystem uint64
		errors         = make(chan error, 1)
	)

	dec := json.NewDecoder(response.Body)

	go func() {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				fmt.Printf("Failed to close body reader: %v", err)
			}
		}(response.Body)
		for {
			var (
				v                 *types.StatsJSON
				memPercent        = 0.0
				cpuPercent        float64
				blkRead, blkWrite uint64 // Only used on Linux
				mem               float64
				memLimit          = 0.0
				pidsStatsCurrent  uint64
			)

			select {
			case <-ctx.Done():
				return
			default:
				//
			}

			if err := dec.Decode(&v); err != nil {
				dec = json.NewDecoder(io.MultiReader(dec.Buffered(), response.Body))
				errors <- err
				if err == io.EOF {
					break
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			daemonOSType := response.OSType

			if daemonOSType != "windows" {
				// MemoryStats.Limit will never be 0 unless the container is not running and we haven't
				// got any Samples from cgroup
				if v.MemoryStats.Limit != 0 {
					memPercent = float64(v.MemoryStats.Usage) / float64(v.MemoryStats.Limit) * 100.0
				}
				previousCPU = v.PreCPUStats.CPUUsage.TotalUsage
				previousSystem = v.PreCPUStats.SystemUsage
				cpuPercent = calculateCPUPercentUnix(previousCPU, previousSystem, v)
				blkRead, blkWrite = calculateBlockIO(v.BlkioStats)
				mem = float64(v.MemoryStats.Usage)
				memLimit = float64(v.MemoryStats.Limit)
				pidsStatsCurrent = v.PidsStats.Current
			} else {
				cpuPercent = calculateCPUPercentWindows(v)
				blkRead = v.StorageStats.ReadSizeBytes
				blkWrite = v.StorageStats.WriteSizeBytes
				mem = float64(v.MemoryStats.PrivateWorkingSet)
			}

			container.mutex.Lock()

			firstSeen := container.Data.LastUpdated == 0
			healthStatusTooOld := time.Since(time.Unix(container.Data.HealthUpdated, 0)) > time.Duration(5)

			container.Data.LastUpdated = v.Stats.Read.Unix()
			container.Data.CpuPercent = cpuPercent
			container.Data.CpuPercentHistory.Add(Sample{float64(container.Data.LastUpdated), cpuPercent})
			container.Data.Memory = uint64(mem)
			container.Data.MemoryPercent = memPercent
			container.Data.MemoryLimit = uint64(memLimit)
			container.Data.MemoryHistory.Add(Sample{float64(container.Data.LastUpdated), mem})
			prevNetworkRx, prevNetworkTx := container.Data.NetworkRx, container.Data.NetworkTx
			container.Data.NetworkRx, container.Data.NetworkTx = calculateNetwork(v.Networks)
			container.Data.NetworkRxHistory.Add(Sample{float64(container.Data.LastUpdated), float64(container.Data.NetworkRx - prevNetworkRx)})
			container.Data.NetworkTxHistory.Add(Sample{float64(container.Data.LastUpdated), float64(container.Data.NetworkTx - prevNetworkTx)})
			container.Data.BlockRead = blkRead
			container.Data.BlockWrite = blkWrite
			container.Data.PIDs = pidsStatsCurrent

			if firstSeen || healthStatusTooOld {
				if inspect, err := cli.ContainerInspect(ctx, container.Data.ID); err == nil {
					if firstSeen {
						for _, env := range inspect.Config.Env {
							if s := strings.SplitN(env, "=", 2); len(s) == 2 {
								container.Data.EnvVars[s[0]] = s[1]
							}
						}
					}
					container.Data.HealthUpdated = container.Data.LastUpdated
					if inspect.State != nil && inspect.State.Health != nil {
						switch inspect.State.Health.Status {
						case "healthy":
							container.Data.HealthStatus = Healthy
						default:
							container.Data.HealthStatus = Unhealthy
						}
					} else {
						container.Data.HealthStatus = UnknownHealth
					}
				} else {
					container.Data.HealthStatus = UnknownHealth
					fmt.Printf("Failed to inspect container %s: %v", container.Data.ID, err)
				}
			}

			stopped := false
			if container.Data.State == ContainerRunning && container.Data.PIDs == 0 {
				// double check that container is still running
				if containers, err := cli.ContainerList(ctx_, types.ContainerListOptions{}); err == nil {
					found := false
					for _, c := range containers {
						if c.ID == container.Data.ID {
							found = true
							break
						}
					}
					if !found {
						stopped = true
						container.Data.State = ContainerStopped
					}
				}
			}

			container.mutex.Unlock()

			if stopped {
				container.OnStopped()
			}

			container.Updated()

			errors <- nil // we just handled a valid update
		}
		fmt.Printf("Done following stats of container %s (%s)\n", container.Data.AlternativeName, container.Data.ID)
	}()
	for {
		select {
		case <-time.After(2 * time.Second):
			fmt.Printf("Timeout while following stats of container %s (%s)\n", container.Data.AlternativeName, container.Data.ID)
		case <-ctx.Done():
			fmt.Printf("Done following stats of container %s (%s)\n", container.Data.AlternativeName, container.Data.ID)
			return
		case err := <-errors:
			if err != nil {
				fmt.Printf("Error while following stats of container %s (%s): %v\n", container.Data.AlternativeName, container.Data.ID, err)
				continue
			}
		}
	}
}

func calculateCPUPercentUnix(previousCPU, previousSystem uint64, v *types.StatsJSON) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.CPUStats.SystemUsage) - float64(previousSystem)
	)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		nofCpu := math.Max(float64(len(v.CPUStats.CPUUsage.PercpuUsage)), float64(v.CPUStats.OnlineCPUs))
		cpuPercent = (cpuDelta / systemDelta) * nofCpu * 100.0
	}
	return cpuPercent
}

func calculateCPUPercentWindows(v *types.StatsJSON) float64 {
	// Max number of 100ns intervals between the previous time read and now
	possIntervals := uint64(v.Read.Sub(v.PreRead).Nanoseconds()) // Start with number of ns intervals
	possIntervals /= 100                                         // Convert to number of 100ns intervals
	possIntervals *= uint64(v.NumProcs)                          // Multiple by the number of processors

	// Intervals used
	intervalsUsed := v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage

	// Percentage avoiding divide-by-zero
	if possIntervals > 0 {
		return float64(intervalsUsed) / float64(possIntervals) * 100.0
	}
	return 0.00
}

func calculateBlockIO(blkio types.BlkioStats) (blkRead uint64, blkWrite uint64) {
	for _, bioEntry := range blkio.IoServiceBytesRecursive {
		switch strings.ToLower(bioEntry.Op) {
		case "read":
			blkRead = blkRead + bioEntry.Value
		case "write":
			blkWrite = blkWrite + bioEntry.Value
		}
	}
	return
}

func calculateNetwork(network map[string]types.NetworkStats) (uint64, uint64) {
	var rx, tx uint64

	for _, v := range network {
		rx += v.RxBytes
		tx += v.TxBytes
	}
	return rx, tx
}
