package main

import (
	"github.com/AllenDang/giu"
	"image"
	"image/color"
)

var _ giu.Widget = &BarWidget{}

// BarWidget Renders horizontal / vertical bar
type BarWidget struct {
	horizontal bool
	foreground color.RGBA
	background color.RGBA
	width      float32
	height     float32
	min        float64
	value      float64
	max        float64
	label      string
}

// Bar creates BarWidget.
func Bar() *BarWidget {
	return &BarWidget{
		horizontal: true,
		foreground: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		background: color.RGBA{R: 70, G: 70, B: 70, A: 255},
		width:      -1,
		height:     -1,
		min:        0,
		max:        1,
	}
}

// Min Set minimum range for value
func (w *BarWidget) Min(min float64) *BarWidget {
	w.min = min
	return w
}

// Value Set value
func (w *BarWidget) Value(value float64) *BarWidget {
	w.value = value
	return w
}

// Max Set maximum range for value
func (w *BarWidget) Max(max float64) *BarWidget {
	w.max = max
	return w
}

// Width Force width of widget
func (w *BarWidget) Width(width float32) *BarWidget {
	w.width = width
	return w
}

// Width Force height of widget
func (w *BarWidget) Height(height float32) *BarWidget {
	w.height = height
	return w
}

// Foreground Set foreground color of widget
func (w *BarWidget) Foreground(color color.RGBA) *BarWidget {
	w.foreground = color
	return w
}

// Background Set foreground color of widget
func (w *BarWidget) Background(color color.RGBA) *BarWidget {
	w.background = color
	return w
}

// Label Set label of widget
func (w *BarWidget) Label(label string) *BarWidget {
	w.label = label
	return w
}

// Build implements Widget interface.
func (w *BarWidget) Build() {
	availWidth, availHeight := giu.GetAvailableRegion()
	var width = w.width
	if width <= 0 {
		width = availWidth
	}
	var height = w.height
	if height <= 0 {
		height = availHeight
	}

	canvas := giu.GetCanvas()
	topLeftPos := giu.GetCursorScreenPos()

	ratio := (w.value - w.min) / (w.max - w.min)
	if w.horizontal {
		canvas.AddQuadFilled(
			topLeftPos,
			topLeftPos.Add(image.Pt(int(width), 0)),
			topLeftPos.Add(image.Pt(int(width), int(height))),
			topLeftPos.Add(image.Pt(0, int(height))),
			w.background,
		)
		canvas.AddQuadFilled(
			topLeftPos,
			topLeftPos.Add(image.Pt(int(float64(width)*ratio), 0)),
			topLeftPos.Add(image.Pt(int(float64(width)*ratio), int(height))),
			topLeftPos.Add(image.Pt(0, int(height))),
			w.foreground,
		)
	}

	// calc height of text to center it vertically in the bar
	_, valueHeight := giu.CalcTextSize(w.label)

	// clip the text rendering at bounds of bar
	giu.PushClipRect(topLeftPos, topLeftPos.Add(image.Pt(int(width), int(height))), false)
	canvas.AddText(topLeftPos.Add(image.Pt(0, int(valueHeight/4))), color.White, w.label)
	giu.PopClipRect()

	giu.Dummy(width, height).Build()
}
