package main

import (
	"fmt"
	"github.com/AllenDang/giu"
	"github.com/AllenDang/imgui-go"
	"image"
	"image/color"
	"math"
)

const (
	DegToRad = 0.017453292519943295769236907684886127134428718885417 // N[Pi/180, 50]
	RadToDeg = 57.295779513082320876798154814105170332405472466564   // N[180/Pi, 50]
)

var _ giu.Widget = &LabeledGaugeWidget{}

// LabeledGaugeWidget Renders a id and a gauge
type LabeledGaugeWidget struct {
	id     string
	min    float32
	value  float32
	format string
	max    float32
}

// LabeledGauge creates LabeledGaugeWidget.
func LabeledGauge(id string) *LabeledGaugeWidget {
	return &LabeledGaugeWidget{
		id:     id,
		min:    0,
		max:    1,
		format: "%0.1f",
	}
}

// Min Set minimum range for value
func (w *LabeledGaugeWidget) Min(min float32) *LabeledGaugeWidget {
	w.min = min
	return w
}

// Value Set value
func (w *LabeledGaugeWidget) Value(value float32) *LabeledGaugeWidget {
	w.value = value
	return w
}

// Value Set string to format value, e.g. "%0.1f%"
func (w *LabeledGaugeWidget) Format(format string) *LabeledGaugeWidget {
	w.format = format
	return w
}

// Max Set maximum range for value
func (w *LabeledGaugeWidget) Max(max float32) *LabeledGaugeWidget {
	w.max = max
	return w
}

func shortenText(width float32, text string) string {
	cut := 2
	newText := text
	for {
		w, _ := giu.CalcTextSize(newText)
		if w < width {
			return newText
		}
		if len(newText) <= cut {
			return newText
		}
		s := len(text)/2 - cut/2
		newText = text[:s] + "…" + text[s+cut:]
		cut++
	}
}

// Build implements Widget interface.
func (w *LabeledGaugeWidget) Build() {
	width, height := giu.GetAvailableRegion()

	defaultFonts := giu.GetDefaultFonts()
	font := defaultFonts[0].SetSize(12)
	if giu.PushFont(font) {
		defer giu.PopFont()
	}

	label := shortenText(width, w.id)
	labelWidth, labelHeight := giu.CalcTextSize(label)

	textFg := giu.Vec4ToRGBA(imgui.CurrentStyle().GetColor(imgui.StyleColorText))
	gaugeBg := color.RGBA{70, 70, 70, 255}
	currentFg := color.RGBA{255, 255, 255, 255}

	canvas := giu.GetCanvas()
	topLeftPos := giu.GetCursorScreenPos()

	gaugeHeight := height - labelHeight
	gaugeCenterPos := topLeftPos.Add(image.Pt(0, int(labelHeight))).Add(image.Pt(int(width/2), int(gaugeHeight/2)))
	var radius float32
	if width > gaugeHeight {
		radius = gaugeHeight / 1.8
	} else {
		radius = width / 2
	}
	radius *= 0.8
	startAngle := 315 * DegToRad
	endAngle := 45 * DegToRad
	ratio := (w.value - w.min) / (w.max - w.min)
	valueAngle := startAngle - math.Min(1, math.Max(0, float64(ratio)))*(startAngle-endAngle)

	canvas.PathClear()
	canvas.PathArcTo(gaugeCenterPos, radius, float32(startAngle+90*DegToRad), float32(endAngle+90*DegToRad), 64)
	canvas.PathFillConvex(gaugeBg)
	canvas.PathStroke(textFg, true, 2)
	canvas.AddCircleFilled(gaugeCenterPos, 4, currentFg)

	gaugeOtPt := image.Pt(
		int(float64(radius)*math.Sin(valueAngle)+float64(gaugeCenterPos.X)),
		int(float64(radius)*math.Cos(valueAngle)+float64(gaugeCenterPos.Y)),
	)
	canvas.AddLine(gaugeCenterPos, gaugeOtPt, currentFg, 2)

	labelPos := image.Pt(gaugeCenterPos.X-int(labelWidth/2), topLeftPos.Y)
	canvas.AddText(labelPos, textFg, label)

	valueText := fmt.Sprintf(w.format, w.value)
	valueWidth, valueHeight := giu.CalcTextSize(valueText)
	valuePos := image.Pt(gaugeCenterPos.X-int(valueWidth/2), topLeftPos.Y+int(height-valueHeight*1.1))

	canvas.AddText(valuePos, textFg, valueText)
}
