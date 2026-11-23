package ab

import "math"

// defaultConfidenceLevel is used when a confidence level is outside (0, 1).
const defaultConfidenceLevel = 0.95

// zScore returns the two-sided standard normal critical value for the
// confidence level, e.g. 1.96 for 0.95.
func zScore(confidenceLevel float64) float64 {
	if !(confidenceLevel > 0 && confidenceLevel < 1) {
		confidenceLevel = defaultConfidenceLevel
	}
	return math.Sqrt2 * math.Erfinv(confidenceLevel)
}

// wilson returns the Wilson score interval for conversions out of n trials.
// It is well behaved for rates near 0 or 1 and small n, unlike the Wald
// interval.
func wilson(conversions, n int64, z float64) ConfidenceInterval {
	if n <= 0 {
		return ConfidenceInterval{Upper: 1}
	}
	fn := float64(n)
	p := float64(conversions) / fn
	z2 := z * z
	d := 1 + z2/fn
	center := (p + z2/(2*fn)) / d
	margin := z * math.Sqrt(p*(1-p)/fn+z2/(4*fn*fn)) / d
	return ConfidenceInterval{Lower: math.Max(0, center-margin), Upper: math.Min(1, center+margin)}
}
