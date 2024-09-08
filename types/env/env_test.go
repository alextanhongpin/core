package env_test

import (
	"testing"

	"github.com/alextanhongpin/core/types/env"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	t.Setenv("STRING", "hello")
	t.Setenv("INT", "10")
	t.Setenv("FLOAT", "10.5")
	t.Setenv("BOOL", "true")

	is := assert.New(t)
	is.Equal("hello", env.Load[string]("STRING"))
	is.Equal(10, env.Load[int]("INT"))
	is.Equal(10.5, env.Load[float64]("FLOAT"))
	is.Equal(true, env.Load[bool]("BOOL"))
	is.PanicsWithError(`env: "UNKNOWN" not set`, func() {
		env.Load[string]("UNKNOWN")
	})
}

func TestLoadSlice(t *testing.T) {
	t.Setenv("STRING", "hello")
	t.Setenv("STRINGS", "hello, world")
	t.Setenv("INTS", "10 20 30")
	t.Setenv("FLOATS", "1.1 2.2 3.3")
	t.Setenv("BOOLS", "true false 1 0 T F")

	is := assert.New(t)
	is.Equal([]string{"hello"}, env.LoadSlice[string]("STRING", " "))
	is.Equal([]string{"hello", "world"}, env.LoadSlice[string]("STRINGS", ","))
	is.Equal([]int{10, 20, 30}, env.LoadSlice[int]("INTS", " "))
	is.Equal([]float64{1.1, 2.2, 3.3}, env.LoadSlice[float64]("FLOATS", " "))
	is.Equal([]bool{true, false, true, false, true, false}, env.LoadSlice[bool]("BOOLS", " "))
}
