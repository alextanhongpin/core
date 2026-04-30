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
