package main

import (
	"math"
	"time"
)

type Sample struct {
	timestamp float64
	value     float64
}

type History struct {
	Samples []Sample
}

const MaxHistorySamples = 512

func NewHistory() History {
	return History{Samples: make([]Sample, 0)}
}

// Add a new sample at the end of history and maintain max history size
func (h *History) Add(sample Sample) {
	if !time.Unix(int64(sample.timestamp), 0).IsZero() {
		h.Samples = append(h.Samples, sample)
		if len(h.Samples) > MaxHistorySamples {
			copy(h.Samples[0:], h.Samples[len(h.Samples)-MaxHistorySamples:])
			h.Samples = h.Samples[:MaxHistorySamples]
		}
	}
}

// GetXY return history samples as two arrays for x and y data
func (h *History) GetXY() ([]float64, []float64) {
	x := make([]float64, len(h.Samples))
	y := make([]float64, len(h.Samples))
	for idx, sample := range h.Samples {
		x[idx] = sample.timestamp
		y[idx] = sample.value
	}
	return x, y
}

// GetYMinMax returns the minimum and maximum value found in history
func (h *History) GetYMinMax(from, until float64) (float64, float64) {
	min := math.MaxFloat64
	max := -math.MaxFloat64
	for _, sample := range h.Samples {
		if sample.timestamp >= from && sample.timestamp <= until {
			max = math.Max(max, sample.value)
			min = math.Min(min, sample.value)
		}
	}
	return min, max
}

func (h *History) GetYAvg(from, until float64) float64 {
	avg := 0.
	samples := 0
	for _, sample := range h.Samples {
		if sample.timestamp >= from && sample.timestamp <= until {
			avg += sample.value
			samples++
		}
	}
	avg /= float64(samples)
	return avg
}
