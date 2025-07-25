package main

import (
	"bytes"
	"fmt"
	"github.com/AllenDang/giu"
	"github.com/inhies/go-bytesize"
	"golang.design/x/clipboard"
	"image/color"
	"image/png"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"
)

const (
	KByte = 1024.0
	MByte = KByte * 1024
	GByte = MByte * 1024

	RecentDuration = time.Duration(5) * time.Minute

	IconSize = 24

	TooltipWidth  = 300
	TooltipHeight = 200
)

type ContainerSortMode int32

const (
	ContainerSortByName    ContainerSortMode = iota
	ContainerSortByCreated ContainerSortMode = iota
)

var (
	LabelColor = color.RGBA{170, 170, 255, 255}

	MemoryIntervals = []float64{64 * KByte, 128 * KByte, 256 * KByte, MByte, 4 * MByte, 8 * MByte, 16 * MByte, 64 * MByte, 256 * MByte, 1 * GByte}
	MemBarColor     = color.RGBA{B: 255, A: 255}

	CpuMaxPercent        = float64(runtime.NumCPU() * 100)
	CpuIntervals         = []float64{1, 5, 10, 25, 50, 100, 200}
	CpuBarColor          = color.RGBA{G: 255, A: 255}
	CpuThrottledBarColor = color.RGBA{R: 255, G: 160, A: 255}
)

// buildPlotTicker returns plot tickers derived from given min, max & interval
func buildPlotTicker(min float64, max float64, interval float64, tickerLabel func(float64) string) []giu.PlotTicker {
	minValue := math.Floor(min/interval) * interval
	maxValue := math.Ceil(max/interval) * interval
	var ticks []giu.PlotTicker
	if minValue == maxValue {
		ticks = make([]giu.PlotTicker, 1)
	} else {
		ticks = make([]giu.PlotTicker, int((maxValue-minValue)/interval)+1)
	}
	for tIdx := range ticks {
		ticks[tIdx].Position = minValue + interval*float64(tIdx)
		ticks[tIdx].Label = tickerLabel(ticks[tIdx].Position)
	}

	return ticks
}

// buildPlotInterval derives an interval from min to max from available intervals
func buildPlotInterval(min, max float64, intervals []float64) float64 {
	diff := max - min
	interval := intervals[0]
	for _, i := range intervals[1:] {
		if diff > 5*interval {
			interval = i
		}
	}
	return interval
}

type App struct {
	containerDataMutex  sync.Mutex
	containerData       []ContainerData
	containerSortMode   ContainerSortMode
	containerIdSelected string

	wnd *giu.MasterWindow

	buildInfo string

	aboutPopup            *PopupModal
	containerEnvVarsPopup *PopupModal
	containerEnvVars      map[string]string

	healthyTexture   *giu.Texture
	unhealthyTexture *giu.Texture
	unknownTexture   *giu.Texture
	restartTexture   *giu.Texture
	stopTexture      *giu.Texture

	stopContainer    func(id string)
	restartContainer func(id string)
}

func NewApp() *App {
	app := &App{containerSortMode: ContainerSortByName}
	app.wnd = giu.NewMasterWindow("Container HUD", 600, 600, 0)
	app.aboutPopup = NewPopupModal("About")
	app.containerIdSelected = ""
	app.containerEnvVars = make(map[string]string, 0)
	app.containerEnvVarsPopup = NewPopupModal("Environment Variables")
	app.buildTextures()
	return app
}

func (a *App) BuildInfo(info string) {
	a.buildInfo = info
}

func (a *App) buildTextures() {
	image, _ := png.Decode(bytes.NewReader(heartHealthyIconData))
	giu.EnqueueNewTextureFromRgba(image, func(tex *giu.Texture) {
		a.healthyTexture = tex
	})

	image, _ = png.Decode(bytes.NewReader(heartUnhealthyIconData))
	giu.EnqueueNewTextureFromRgba(image, func(tex *giu.Texture) {
		a.unhealthyTexture = tex
	})

	image, _ = png.Decode(bytes.NewReader(heartUnknownIconData))
	giu.EnqueueNewTextureFromRgba(image, func(tex *giu.Texture) {
		a.unknownTexture = tex
	})

	image, _ = png.Decode(bytes.NewReader(restartIconData))
	giu.EnqueueNewTextureFromRgba(image, func(tex *giu.Texture) {
		a.restartTexture = tex
	})

	image, _ = png.Decode(bytes.NewReader(stopIconData))
	giu.EnqueueNewTextureFromRgba(image, func(tex *giu.Texture) {
		a.stopTexture = tex
	})
}

func (a *App) OnStopContainer(stopContainer func(id string)) *App {
	a.stopContainer = stopContainer
	return a
}

func (a *App) OnRestartContainer(restartContainer func(id string)) *App {
	a.restartContainer = restartContainer
	return a
}

func (a *App) ContainerData(data []ContainerData) {
	a.containerDataMutex.Lock()
	defer a.containerDataMutex.Unlock()

	a.containerData = data

	giu.Update()
}

func (a *App) setContainerSelectedByIdx(idx int) {
	selectedId := ""
	if idx >= -1 && idx < len(a.containerData) {
		selectedId = a.containerData[idx].ID
	}
	if selectedId != a.containerIdSelected {
		a.containerIdSelected = selectedId
		giu.Update()
	}
}

func (a *App) IsContainerSelected() bool {
	return a.getContainerByIdx(a.containerIdSelected) >= 0
}

func (a *App) getContainerByIdx(id string) int {
	for i := range a.containerData {
		if a.containerData[i].ID == id {
			return i
		}
	}
	return -1
}

func (a *App) showContainerEnvVars(containerId string) {
	idx := a.getContainerByIdx(containerId)
	if idx >= 0 && idx < len(a.containerData) {
		app.containerEnvVars = make(map[string]string)
		for k, v := range a.containerData[idx].EnvVars {
			app.containerEnvVars[k] = v
		}
		app.containerEnvVarsPopup.Open()
		return
	}
}

func (a *App) sortContainerData() {
	sort.SliceStable(a.containerData, func(i int, j int) bool {
		switch a.containerSortMode {
		case ContainerSortByCreated:
			if a.containerData[i].Created < a.containerData[j].Created {
				return true
			}
			if a.containerData[i].Created == a.containerData[j].Created {
				return a.containerData[i].Name > a.containerData[j].Name
			}
		case ContainerSortByName:
			if a.containerData[i].AlternativeName < a.containerData[j].AlternativeName {
				return true
			}
			if a.containerData[i].AlternativeName == a.containerData[j].AlternativeName {
				return a.containerData[i].Name > a.containerData[j].Name
			}
		}
		return false
	})
}

// Run the main loop, will return when application got closed
func (a *App) Run() {
	a.wnd.Run(a.Render)
}

// Render builds the UI
func (a *App) Render() {
	a.containerDataMutex.Lock()
	defer a.containerDataMutex.Unlock()

	nofContainer := len(a.containerData)
	totalCpuPercent := float64(0)
	totalMemory := uint64(0)
	for _, data := range a.containerData {
		totalCpuPercent += data.CpuPercent
		totalMemory += data.Memory
	}

	w, h := app.wnd.GetSize()
	bestColumns := 0
	bestRows := 0
	bestFit := math.MaxFloat64
	targetAspectRatio := 1.
	for columns := 1; columns <= nofContainer; columns++ {
		rows := int(math.Ceil(float64(nofContainer) / float64(columns)))
		fit := math.Abs(targetAspectRatio - (float64(w)/float64(columns))/(float64(h)/float64(rows)))
		if fit < bestFit {
			bestColumns = columns
			bestRows = rows
			bestFit = fit
		}
	}

	a.sortContainerData()

	giu.SingleWindowWithMenuBar().Layout(
		app.aboutPopup.Layout(
			giu.Label(a.buildInfo),
		),
		giu.PrepareMsgbox(),

		app.containerEnvVarsPopup.Layout(
			giu.Label("Click variable to copy it to clipboard..."),
			giu.Custom(func() {
				envVarNames := make([]string, 0, len(app.containerEnvVars))
				for n, _ := range app.containerEnvVars {
					envVarNames = append(envVarNames, n)
				}
				sort.SliceStable(envVarNames, func(i int, j int) bool {
					return envVarNames[i] < envVarNames[j]
				})
				for _, n := range envVarNames {
					env := fmt.Sprintf("%s=%s", n, app.containerEnvVars[n])
					giu.Selectable(env).OnClick(
						func() {
							fmt.Printf("Copied envvar %s to clipboard\n", n)
							clipboard.Write(clipboard.FmtText, []byte(env))
						}).Build()
				}
			}),
		),
		giu.PrepareMsgbox(),

		giu.MenuBar().Layout(
			giu.Menu("View").Layout(
				giu.MenuItem("Sort containers by name").Selected(a.containerSortMode == ContainerSortByName).OnClick(func() {
					a.containerSortMode = ContainerSortByName
				}),
				giu.MenuItem("Sort containers by creation time").Selected(a.containerSortMode == ContainerSortByCreated).OnClick(func() {
					a.containerSortMode = ContainerSortByCreated
				}),
			),
			giu.Menu("Container").Enabled(a.IsContainerSelected()).Layout(
				giu.MenuItem("Show envvars").OnClick(func() {
					a.showContainerEnvVars(a.containerIdSelected)
				}),
			),
			giu.Menu("Help").Layout(
				giu.MenuItem("About").OnClick(func() {
					a.aboutPopup.Open()
				}),
			),
		),
		giu.Condition(
			nofContainer > 0,
			giu.Layout{
				giu.Row(
					giu.Label(
						fmt.Sprintf("%d containers ( %5.1f%% CPU, %s Mem )",
							nofContainer,
							totalCpuPercent,
							bytesize.New(float64(totalMemory)).String(),
						),
					),
				),
			},
			giu.Layout{
				giu.Label("No containers are running"),
			},
		),
		GridBuilder[ContainerData]("containers", bestColumns, bestRows, a.containerData, a.getContainerByIdx(a.containerIdSelected),
			a.setContainerSelectedByIdx,
			func(_ int, selected bool, data ContainerData) giu.Widget {
				return a.renderContainerData(selected, data)
			}),
	)
}

func (a *App) renderContainerData(selected bool, data ContainerData) giu.Widget {
	minXAxis := float64(time.Now().Add(-RecentDuration).Unix())
	maxXAxis := float64(time.Now().Unix())
	return giu.Layout([]giu.Widget{
		conditionalTexture(data.HealthStatus == UnknownHealth, a.unknownTexture, "Unknown container health status"),
		conditionalTexture(data.HealthStatus == Unhealthy, a.unhealthyTexture, "Container is unhealthy"),
		conditionalTexture(data.HealthStatus == Healthy, a.healthyTexture, "Container is healthy"),
		giu.Custom(func() { giu.SameLine() }),
		conditionalButton(data.State == ContainerRunning, a.restartTexture, "Restart container", func() {
			go a.restartContainer(data.ID)
		}),
		giu.Custom(func() { giu.SameLine() }),
		conditionalButton(data.State == ContainerRunning, a.stopTexture, "Stop container", func() {
			go a.stopContainer(data.ID)
		}),
		giu.Dummy(0, 0),
		ShortLabel(data.AlternativeName),
		giu.ContextMenu().Layout(
			giu.Label(fmt.Sprintf("Uptime %s", time.Since(time.Unix(data.Created, 0)).Round(time.Second))),
			giu.Label(fmt.Sprintf("Image  %s", data.Image)),
		),
		ShortLabel(fmt.Sprintf("ID %s", data.ID[:12])),
		giu.ContextMenu().Layout(
			giu.Selectable("Copy to clipboard").OnClick(func() {
				fmt.Printf("Copied ID %s to clipboard\n", data.ID[:12])
				clipboard.Write(clipboard.FmtText, []byte(data.ID[:12]))
			}),
		),
		giu.Column(
			Bar().Label(
				fmt.Sprintf("CPU  %0.1f%%, %d PIDs", data.CpuPercent, data.PIDs),
			).Min(0).Value(data.CpuPercent).Max(CpuMaxPercent).Height(16).Foreground(CpuBarColor),
			Bar().Label(
				fmt.Sprintf("     %0.1f%% throttled", data.CpuThrottledPercent),
			).Min(0).Value(data.CpuThrottledPercent).Max(100).Height(16).Foreground(CpuThrottledBarColor),
		),
		cpuHistoryTooltip(data, minXAxis, maxXAxis),
		Bar().Label(
			fmt.Sprintf("Mem  %0.1f%% = %s", data.MemoryPercent, bytesize.New(float64(data.Memory))),
		).Min(0).Value(float64(data.Memory)).Max(float64(data.MemoryLimit)).Height(16).Foreground(MemBarColor),
		memoryHistoryTooltip(data, minXAxis, maxXAxis),
		giu.Label(fmt.Sprintf("Network RX %s\n        TX %s", bytesize.New(float64(data.NetworkRx)), bytesize.New(float64(data.NetworkTx)))),
		networkHistoryTooltip(data, minXAxis, maxXAxis),
	})
}

func conditionalTexture(show bool, texture *giu.Texture, tooltip string) giu.Widget {
	return giu.Condition(
		show,
		giu.Layout{
			giu.Image(texture).BorderCol(color.Transparent).Size(IconSize, IconSize),
			giu.Tooltip(tooltip),
		},
		nil,
	)
}

func conditionalButton(show bool, texture *giu.Texture, tooltip string, onClick func()) giu.Widget {
	return giu.Condition(
		show,
		giu.Layout{
			giu.ImageButton(texture).FramePadding(0).BgColor(color.Transparent).Size(IconSize, IconSize).OnClick(onClick),
			giu.Tooltip(tooltip),
		},
		nil,
	)
}

func cpuHistoryTooltip(data ContainerData, minXAxis float64, maxXAxis float64) giu.Widget {
	return giu.Tooltip("CPU History").Layout(
		giu.Label(data.AlternativeName),
		giu.Custom(func() {
			var (
				xTicks                                            []giu.PlotTicker = nil
				yTicks                                            []giu.PlotTicker = nil
				yAxisMin                                                           = 0.
				yAxisMax                                                           = 0.
				cpuX, cpuY                                                         = make([]float64, 0), make([]float64, 0)
				cpuThrottledX, cpuThrottledY                                       = make([]float64, 0), make([]float64, 0)
				cpuMin, cpuMax, cpuAvg                            float64
				cpuThrottledMin, cpuThrottledMax, cpuThrottledAvg float64
				totalMin, totalMax                                float64
			)
			if len(data.CpuPercentHistory.Samples) > 0 {
				cpuX, cpuY = data.CpuPercentHistory.GetXY()
				cpuMin, cpuMax = data.CpuPercentHistory.GetYMinMax(minXAxis, maxXAxis)
				cpuAvg = data.CpuPercentHistory.GetYAvg(minXAxis, maxXAxis)
				cpuThrottledX, cpuThrottledY = data.CpuThrottledPercentHistory.GetXY()
				cpuThrottledMin, cpuThrottledMax = data.CpuThrottledPercentHistory.GetYMinMax(minXAxis, maxXAxis)
				cpuThrottledAvg = data.CpuThrottledPercentHistory.GetYAvg(minXAxis, maxXAxis)
				totalMin = math.Min(cpuMin, cpuThrottledMin)
				totalMax = math.Max(cpuMax, cpuThrottledMax)
				xTicks = buildPlotTicker(minXAxis, maxXAxis, 2*time.Minute.Seconds(), func(value float64) string {
					h, m, _ := time.Unix(int64(value), 0).Clock()
					return fmt.Sprintf("%02d:%02d", h, m)
				})
				yInterval := buildPlotInterval(totalMin, totalMax, CpuIntervals)
				yTicks = buildPlotTicker(totalMin, totalMax, yInterval, func(value float64) string {
					return fmt.Sprintf("%0.0f %%", value)
				})
				yAxisMin = yTicks[0].Position
				yAxisMax = yTicks[len(yTicks)-1].Position
			}
			giu.Plot(
				fmt.Sprintf("CPU: avg %0.1f%%, max %0.1f%%,\navg throttled %0.1f%%", cpuAvg, cpuMax, cpuThrottledAvg),
			).Size(
				TooltipWidth, TooltipHeight,
			).AxisLimits(
				minXAxis,
				maxXAxis,
				yAxisMin,
				yAxisMax,
				giu.ConditionAlways,
			).Plots(
				giu.PlotLineXY("CPU", cpuX, cpuY),
				giu.PlotLineXY("Throttled", cpuThrottledX, cpuThrottledY),
			).XTicks(
				xTicks, false,
			).YTicks(
				yTicks, false, 0,
			).XAxeFlags(
				giu.PlotAxisFlagsTime,
			).Build()
		}),
	)
}

func memoryHistoryTooltip(data ContainerData, minXAxis float64, maxXAxis float64) giu.Widget {
	return giu.Tooltip("Mem History").Layout(
		giu.Label(data.AlternativeName),
		giu.Custom(func() {
			var (
				xTicks                 []giu.PlotTicker = nil
				yTicks                 []giu.PlotTicker = nil
				yAxisMin                                = 0.
				yAxisMax                                = 0.
				memX, memY                              = make([]float64, 0), make([]float64, 0)
				memMin, memMax, memAvg float64
			)
			if len(data.MemoryHistory.Samples) > 0 {
				memX, memY = data.MemoryHistory.GetXY()
				memMin, memMax = data.MemoryHistory.GetYMinMax(minXAxis, maxXAxis)
				memAvg = data.MemoryHistory.GetYAvg(minXAxis, maxXAxis)
				xTicks = buildPlotTicker(minXAxis, maxXAxis, 2*time.Minute.Seconds(), func(value float64) string {
					h, m, _ := time.Unix(int64(value), 0).Clock()
					return fmt.Sprintf("%02d:%02d", h, m)
				})
				yInterval := buildPlotInterval(memMin, memMax, MemoryIntervals)
				yTicks = buildPlotTicker(memMin, memMax, yInterval, func(value float64) string {
					return bytesize.New(value).String()
				})
				yAxisMin = yTicks[0].Position
				yAxisMax = yTicks[len(yTicks)-1].Position
			}
			giu.Plot(
				fmt.Sprintf("Mem: avg %s\n     max %s", bytesize.New(memAvg).String(), bytesize.New(memMax).String()),
			).Size(
				TooltipWidth, TooltipHeight,
			).Flags(
				giu.PlotFlagsNoLegend,
			).AxisLimits(
				minXAxis,
				maxXAxis,
				yAxisMin,
				yAxisMax,
				giu.ConditionAlways,
			).Plots(
				giu.PlotLineXY("Mem", memX, memY),
			).XAxeFlags(
				giu.PlotAxisFlagsTime,
			).XTicks(
				xTicks, false,
			).YTicks(
				yTicks, false, 0,
			).Build()
		}),
	)
}

func networkHistoryTooltip(data ContainerData, minXAxis float64, maxXAxis float64) giu.Widget {
	return giu.Tooltip("Network History").Layout(
		giu.Label(data.AlternativeName),
		giu.Custom(func() {
			var (
				xTicks                       []giu.PlotTicker = nil
				yTicks                       []giu.PlotTicker = nil
				yAxisMin                                      = 0.
				yAxisMax                                      = 0.
				netRxX, netRxY                                = make([]float64, 0), make([]float64, 0)
				netTxX, netTxY                                = make([]float64, 0), make([]float64, 0)
				netRxMin, netRxMax, netRxAvg float64
				netTxMin, netTxMax, netTxAvg float64
			)
			if !time.Unix(data.LastUpdated, 0).IsZero() && len(data.NetworkRxHistory.Samples) > 0 {
				netRxX, netRxY = data.NetworkRxHistory.GetXY()
				netRxMin, netRxMax = data.NetworkRxHistory.GetYMinMax(minXAxis, maxXAxis)
				netRxAvg = data.NetworkRxHistory.GetYAvg(minXAxis, maxXAxis)
				netTxX, netTxY = data.NetworkTxHistory.GetXY()
				netTxMin, netTxMax = data.NetworkTxHistory.GetYMinMax(minXAxis, maxXAxis)
				netTxAvg = data.NetworkTxHistory.GetYAvg(minXAxis, maxXAxis)
				netMin := math.Min(netRxMin, netTxMin)
				netMax := math.Max(netRxMax, netTxMax)
				xTicks = buildPlotTicker(minXAxis, maxXAxis, 2*time.Minute.Seconds(), func(value float64) string {
					h, m, _ := time.Unix(int64(value), 0).Clock()
					return fmt.Sprintf("%02d:%02d", h, m)
				})
				yInterval := buildPlotInterval(netMin, netMax, MemoryIntervals)
				yTicks = buildPlotTicker(netMin, netMax, yInterval, func(value float64) string {
					return bytesize.New(value).String()
				})
				yAxisMin = yTicks[0].Position
				yAxisMax = yTicks[len(yTicks)-1].Position
			}
			giu.Plot(
				fmt.Sprintf(
					"RX: avg %s, max %s\nTX: avg %s, max %s",
					bytesize.New(netRxAvg).String(), bytesize.New(netRxMax).String(),
					bytesize.New(netTxAvg).String(), bytesize.New(netTxMax).String()),
			).Size(
				TooltipWidth, TooltipHeight,
			).AxisLimits(
				minXAxis,
				maxXAxis,
				yAxisMin,
				yAxisMax,
				giu.ConditionAlways,
			).Plots(
				giu.PlotLineXY("RX", netRxX, netRxY),
				giu.PlotLineXY("TX", netTxX, netTxY),
			).XAxeFlags(
				giu.PlotAxisFlagsTime,
			).XTicks(
				xTicks, false,
			).YTicks(
				yTicks, false, 0,
			).Build()
		}),
	)
}
