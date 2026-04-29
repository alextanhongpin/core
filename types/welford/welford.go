package welford

import (
	"math"
)

// WelfordAnomalyDetector tracks running stats to find outliers.
type WelfordAnomalyDetector struct {
	n         int
	mean      float64
	m2        float64
	threshold float64
}

// NewDetector initializes a detector with a specific Z-score threshold.
func NewDetector(threshold float64) *WelfordAnomalyDetector {
	return &WelfordAnomalyDetector{
		threshold: threshold,
	}
}

// Update processes a new value and returns true if it is an anomaly.
func (w *WelfordAnomalyDetector) Update(x float64) bool {
	w.n++
	delta := x - w.mean
	w.mean += delta / float64(w.n)
	delta2 := x - w.mean
	w.m2 += delta * delta2

	if w.n < 2 {
		return false // Need at least two points for variance
	}

	variance := w.m2 / float64(w.n-1)
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return false
	}

	zScore := math.Abs(x-w.mean) / stdDev
	return zScore > w.threshold
}
