package env_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/env"
	"github.com/stretchr/testify/assert"
)

func TestMustLoad(t *testing.T) {
	t.Setenv("STRING", "hello")
	t.Setenv("INT", "10")
	t.Setenv("FLOAT", "10.5")
	t.Setenv("BOOL", "true")

	is := assert.New(t)
	is.Equal("hello", env.MustLoad[string]("STRING"))
	is.Equal(10, env.MustLoad[int]("INT"))
	is.Equal(10.5, env.MustLoad[float64]("FLOAT"))
	is.Equal(true, env.MustLoad[bool]("BOOL"))
	is.Panics(func() {
		env.MustLoad[string]("UNKNOWN")
	})
}

func TestMustLoadSlice(t *testing.T) {
	t.Setenv("STRING", "hello")
	t.Setenv("STRINGS", "hello, world")
	t.Setenv("INTS", "10 20 30")
	t.Setenv("FLOATS", "1.1 2.2 3.3")
	t.Setenv("BOOLS", "true false 1 0 T F")

	is := assert.New(t)
	is.Equal([]string{"hello"}, env.MustLoadSlice[string]("STRING", " "))
	is.Equal([]string{"hello", "world"}, env.MustLoadSlice[string]("STRINGS", ","))
	is.Equal([]int{10, 20, 30}, env.MustLoadSlice[int]("INTS", " "))
	is.Equal([]float64{1.1, 2.2, 3.3}, env.MustLoadSlice[float64]("FLOATS", " "))
	is.Equal([]bool{true, false, true, false, true, false}, env.MustLoadSlice[bool]("BOOLS", " "))
}

func TestMustLoadDuration(t *testing.T) {
	t.Setenv("DURATION_ZERO", "0")
	t.Setenv("DURATION_SECONDS", "10s")

	is := assert.New(t)
	is.Equal(0*time.Second, env.MustLoadDuration("DURATION_ZERO"))
	is.Equal(10*time.Second, env.MustLoadDuration("DURATION_SECONDS"))
}

func TestMustLoadTime(t *testing.T) {
	t.Setenv("TIME", "2023-10-01T12:00:00Z")
	t.Setenv("DATE", "2023-10-01")
	t.Setenv("TIME_INVALID", "invalid")

	is := assert.New(t)
	expectedTime, err := time.Parse(time.RFC3339, "2023-10-01T12:00:00Z")
	is.NoError(err)
	is.Equal(expectedTime, env.MustLoadTime("TIME", time.RFC3339))

	expectedDate, err := time.Parse("2006-01-02", "2023-10-01")
	is.NoError(err)
	is.Equal(expectedDate, env.MustLoadTime("DATE", "2006-01-02"))

	is.Panics(func() {
		env.MustLoadTime("TIME_INVALID", time.RFC3339)
	})
}
