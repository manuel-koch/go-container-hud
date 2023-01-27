package main

import (
	"github.com/AllenDang/giu"
	"github.com/AllenDang/imgui-go"
	"image"
	"image/color"
)

type GridState struct {
	itemWidth  float32
	itemHeight float32
}

func (s *GridState) Dispose() {
	// Nothing to do here.
}

// GridBuilder Arrange widgets from given array of Samples in a grid layout where all grid items share equal size
// The selected item will be highlighted by a border.
func GridBuilder[T any](id string, columns int, rows int, values []T, selected int, onClicked func(i int), builder func(i int, selected bool, item T) giu.Widget) giu.Layout {
	var layout giu.Layout

	layout = append(layout, giu.Custom(func() {
		imgui.PushID(id)

		w, h := giu.GetAvailableRegion()
		spacingX, spacingY := giu.GetItemSpacing()
		itemWidth := (w - spacingX*float32(columns-1)) / float32(columns)
		itemHeight := (h - spacingY*float32(rows-1)) / float32(rows)

		var state *GridState
		if s := giu.Context.GetState(id); s == nil {
			state = &GridState{itemWidth: itemWidth, itemHeight: itemHeight}
			giu.Context.SetState(id, state)
		} else {
			state = s.(*GridState)
			state.itemWidth = itemWidth
			state.itemHeight = itemHeight
		}
	}))

	if len(values) > 0 && builder != nil {
		for i, v := range values {
			itemIdx := i
			itemColumns := columns
			if i >= columns*rows {
				break
			}
			valueRef := v
			layout = append(layout, giu.Custom(func() {
				_ = itemIdx
				_ = itemColumns
				if itemIdx%itemColumns > 0 {
					giu.SameLine()
				}
				if s := giu.Context.GetState(id); s != nil {
					state := s.(*GridState)
					isSelected := itemIdx == selected
					giu.Child().Border(true).Size(state.itemWidth, state.itemHeight).Flags(giu.WindowFlagsNoScrollbar|giu.WindowFlagsNoScrollWithMouse).Layout(
						giu.Custom(func() {
							if !isSelected {
								return
							}
							style := imgui.CurrentStyle()
							availWidth, availHeight := giu.GetAvailableRegion()
							canvas := giu.GetCanvas()
							topLeftPos := giu.GetCursorScreenPos().Sub(image.Pt(int(style.FramePadding().X), int(2*style.FramePadding().Y)))
							canvas.AddRect(
								topLeftPos,
								topLeftPos.Add(image.Pt(int(availWidth+2*style.FramePadding().X), int(availHeight+4*style.FramePadding().Y))),
								color.RGBA{R: 255, G: 255, B: 255, A: 255},
								6.,
								giu.DrawFlagsRoundCornersAll,
								2.,
							)
						}),
						builder(itemIdx, isSelected, valueRef),
					).Build()
					giu.Event().OnClick(giu.MouseButtonLeft, func() {
						if onClicked != nil {
							onClicked(itemIdx)
						}
					}).Build()
				}
			}))
		}
	}

	layout = append(layout, giu.Custom(func() { imgui.PopID() }))

	return layout
}
