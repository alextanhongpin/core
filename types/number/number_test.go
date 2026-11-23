package number_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/alextanhongpin/core/types/number"
	"github.com/go-openapi/testify/assert"
)

func TestClip(t *testing.T) {
	is := assert.New(t)
	is.Equal(0, number.Clip(0, 10, -5))
	is.Equal(5, number.Clip(0, 10, 5))
	is.Equal(10, number.Clip(0, 10, 15))
	is.Equal(uint8(255), number.Clip[uint8](0, 255, 255))

	// Regression: reversed bounds used to always return hi.
	is.Equal(5, number.Clip(10, 0, 5))
	is.Equal(10, number.Clip(10, 0, 15))
	is.Equal(0, number.Clip(10, 0, -1))

	is.True(math.IsNaN(number.Clip(0, 1, math.NaN())))
}

func TestInRange(t *testing.T) {
	is := assert.New(t)
	is.True(number.InRange(0, 10, 0))
	is.True(number.InRange(0, 10, 10))
	is.False(number.InRange(0, 10, 11))
	is.True(number.InRange(10, 0, 5), "reversed bounds")
	is.False(number.InRange(0, 1, math.NaN()))
}

func TestSign(t *testing.T) {
	is := assert.New(t)
	is.Equal(-1, number.Sign(-3))
	is.Equal(0, number.Sign(0))
	is.Equal(1, number.Sign(2.5))
	is.Equal(0, number.Sign(math.NaN()))
}

func TestMap(t *testing.T) {
	is := assert.New(t)
	is.Equal(180.0, number.Map(400.0, 0, 800, 0, 360))
	is.Equal(90.0, number.Map(25.0, 0, 40, 240, 0), "reversed output range")
	is.Equal(1.5, number.Map(15.0, 0, 10, 0, 1), "unclamped")
	is.Equal(7.0, number.Map(3.0, 5, 5, 7, 9), "empty input range returns outMin")
}

func TestNormalize(t *testing.T) {
	is := assert.New(t)
	is.Equal(0.25, number.Normalize(25.0, 0, 100))
	is.Equal(0.0, number.Normalize(5.0, 5, 5))
}

func TestLerp(t *testing.T) {
	is := assert.New(t)
	is.Equal(34.0, number.Lerp(10, 90, 0.3))
	is.Equal(110.0, number.Lerp(10, 90, 1.25), "extrapolates")
}

func ExampleClip() {
	fmt.Println(number.Clip(0, 100, 120))
	fmt.Println(number.Clip(0.0, 1.0, -0.5))
	// Output:
	// 100
	// 0
}

func ExampleMap() {
	// A 10-bit sensor reading to volts.
	fmt.Printf("%.2f\n", number.Map(512.0, 0, 1023, 0, 5))
	// Output: 2.50
}
