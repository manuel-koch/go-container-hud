package main

import (
	"github.com/AllenDang/giu"
	"github.com/AllenDang/imgui-go"
)

type GridState struct {
	itemWidth  float32
	itemHeight float32
}

func (s *GridState) Dispose() {
	// Nothing to do here.
}

// GridBuilder Arrange widgets from given array of Samples in a grid layout where all grid items share equal size
func GridBuilder[T any](id string, columns int, rows int, values []T, builder func(i int, item T) giu.Widget) giu.Layout {
	var layout giu.Layout
	var foo = 1

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
			//if i%columns > 0 {
			//	layout = append(layout, giu.Custom(func() { giu.SameLine() }))
			//}
			valueRef := v
			layout = append(layout, giu.Custom(func() {
				_ = foo
				_ = itemIdx
				_ = itemColumns
				if itemIdx%itemColumns > 0 {
					giu.SameLine()
				}
				if s := giu.Context.GetState(id); s != nil {
					state := s.(*GridState)
					giu.Child().Border(false).Size(state.itemWidth, state.itemHeight).Layout(
						builder(i, valueRef),
					).Build()
				}
			}))
		}
	}

	layout = append(layout, giu.Custom(func() { imgui.PopID() }))

	return layout
}
