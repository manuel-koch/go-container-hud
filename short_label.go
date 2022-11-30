package main

import "github.com/AllenDang/giu"

type ShortLabelWidget struct {
	label string
}

func ShortLabel(label string) *ShortLabelWidget {
	return &ShortLabelWidget{label: label}
}

func (l *ShortLabelWidget) Build() {
	width, _ := giu.GetAvailableRegion()
	label := shortenText(width, l.label)
	giu.Label(label).Build()
}
