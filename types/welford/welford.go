package welford

import (
	"math"
)

// WelfordAnomalyDetector keeps a running mean and sample variance using
// Welford's online algorithm and flags values whose z-score exceeds a
// threshold. The zero value has threshold 0. It is not safe for concurrent use.
type WelfordAnomalyDetector struct {
	n         int
	mean      float64
	m2        float64
	threshold float64
}

// NewDetector returns a detector that flags values more than threshold
// sample standard deviations from the mean.
func NewDetector(threshold float64) *WelfordAnomalyDetector {
	return &WelfordAnomalyDetector{
		threshold: threshold,
	}
}

// Update scores x against the values seen so far, then adds x to the running
// statistics. It returns true if x is an anomaly. It never flags until at
// least two earlier values exist with nonzero spread.
func (w *WelfordAnomalyDetector) Update(x float64) bool {
	anomaly := false
	if sd := w.StdDev(); sd > 0 {
		anomaly = math.Abs(x-w.mean)/sd > w.threshold
	}

	w.n++
	delta := x - w.mean
	w.mean += delta / float64(w.n)
	w.m2 += delta * (x - w.mean)

	return anomaly
}

// Count returns the number of values added.
func (w *WelfordAnomalyDetector) Count() int { return w.n }

// Mean returns the running mean, or 0 before any value is added.
func (w *WelfordAnomalyDetector) Mean() float64 { return w.mean }

// Variance returns the sample variance, or 0 with fewer than two values.
func (w *WelfordAnomalyDetector) Variance() float64 {
	if w.n < 2 {
		return 0
	}
	return w.m2 / float64(w.n-1)
}

// StdDev returns the sample standard deviation.
func (w *WelfordAnomalyDetector) StdDev() float64 {
	return math.Sqrt(w.Variance())
}
