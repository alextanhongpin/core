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
