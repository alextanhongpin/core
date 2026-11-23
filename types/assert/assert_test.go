package assert

import (
	"testing"

	"github.com/go-openapi/testify/assert"
)

func zeroCase(yield func(any, string) bool) {
	yield("", "empty string")
	yield(0, "zero int")
	yield(0.0, "zero float")
	yield(false, "false bool")
	yield(nil, "nil")
	yield([]int(nil), "nil slice")
	yield([]int{}, "empty slice")
	yield(map[string]any{}, "empty map")

	var m map[string]any
	yield(m, "nil map")

	var a any
	yield(a, "any")

	type T struct{}

	var v T
	yield(v, "zero struct")
	yield(T{}, "zero strut")

	var vp *T
	yield(vp, "nil struct pointer")
	yield(new(T), "nil struct pointer")
}

func TestIsZero(t *testing.T) {
	is := assert.New(t)
	for v, name := range zeroCase {
		is.True(IsZero(v), name)
	}
}

func TestRequired(t *testing.T) {
	is := assert.New(t)
	for v, name := range zeroCase {
		is.Equal("required", Required(v), name)
	}
}

func TestOptional(t *testing.T) {
	is := assert.New(t)
	for v, name := range zeroCase {
		is.Equal("", Optional(v), name)
	}
}

func TestIsZeroArrays(t *testing.T) {
	is := assert.New(t)
	is.True(IsZero([3]int{}), "zero array")
	is.False(IsZero([3]int{1}), "non-zero array")
	is.True(IsZero(new(int)), "pointer to zero")
	n := 1
	is.False(IsZero(&n), "pointer to non-zero")
}

func TestLengthCountsRunes(t *testing.T) {
	is := assert.New(t)
	is.Equal("", MaxLength("héllo", 5))
	is.Equal("", MinLength("日本", 2))
	is.Equal("must be at least 3 characters", MinLength("日本", 3))
}
