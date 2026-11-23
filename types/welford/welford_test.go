package welford_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/welford"
	"github.com/go-openapi/testify/assert"
)

func TestWelford(t *testing.T) {
	is := assert.New(t)

	detector := welford.NewDetector(2.0)
	data := []float64{10, 12, 11, 13, 10, 12, 10, 9, 10, 13, 11, 100, 12, 11}

	for _, val := range data {
		isAnomaly := detector.Update(val)
		is.Equal(val == 100.0, isAnomaly)
	}
}

func TestStats(t *testing.T) {
	is := assert.New(t)

	d := welford.NewDetector(3)
	is.Zero(d.Variance())
	for _, v := range []float64{2, 4, 4, 4, 5, 5, 7, 9} {
		d.Update(v)
	}
	is.Equal(8, d.Count())
	is.InDelta(5.0, d.Mean(), 1e-9)
	is.InDelta(32.0/7, d.Variance(), 1e-9)
}

// The value under test must not be part of the statistics it is scored
// against; otherwise a single large outlier inflates the variance and hides
// itself.
func TestOutlierScoredAgainstPriorValues(t *testing.T) {
	is := assert.New(t)

	d := welford.NewDetector(3)
	d.Update(10)
	d.Update(11)
	is.True(d.Update(20))
}
