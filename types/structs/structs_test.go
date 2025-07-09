package structs_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/structs"
	"github.com/stretchr/testify/assert"
)

type Time struct {
}

func TestType(t *testing.T) {
	now := time.Now()
	is := assert.New(t)
	is.Equal("time.Time", structs.Type(now))
	is.Equal("*time.Time", structs.Type(&now))

	var clock Time
	is.Equal("structs_test.Time", structs.Type(clock))
	is.Equal("*structs_test.Time", structs.Type(&clock))

	is.Equal("nil", structs.Type(nil))
	is.Equal("string", structs.Type(""))
}

func TestName(t *testing.T) {
	now := time.Now()

	is := assert.New(t)
	is.Equal("time.Time", structs.PkgName(now))
	is.Equal("Time", structs.Name(now))

	var clock Time
	is.Equal("structs_test.Time", structs.PkgName(clock))
	is.Equal("Time", structs.Name(clock))
}

func TestNonZero(t *testing.T) {
	type User struct {
		Name string
	}

	t.Run("empty struct", func(t *testing.T) {
		var u User
		err := structs.NonZero(u)

		is := assert.New(t)
		var fieldErr *structs.FieldError
		is.True(errors.As(err, &fieldErr))
		is.Equal(fieldErr.Path, "structs_test.User.Name")
		is.Equal(fieldErr.Field, "Name")
	})

	t.Run("nil struct", func(t *testing.T) {
		var u *User
		err := structs.NonZero(u)

		is := assert.New(t)
		var fieldErr *structs.FieldError
		is.True(errors.As(err, &fieldErr))
		is.Equal(fieldErr.Path, "structs_test.User")
		is.Equal(fieldErr.Field, "User")
	})

	t.Run("filled struct", func(t *testing.T) {
		u := User{Name: "Jon Appleseed"}
		is := assert.New(t)
		is.Nil(structs.NonZero(u))
	})

	t.Run("nil", func(t *testing.T) {
		err := structs.NonZero(nil)
		is := assert.New(t)
		is.Error(err)

		var fieldErr *structs.FieldError
		is.True(errors.As(err, &fieldErr))
		is.Equal(fieldErr.Path, "nil")
		is.Equal(fieldErr.Field, "nil")
	})
}

func TestIsNil(t *testing.T) {
	is := assert.New(t)

	// nil interface
	var v any = nil
	is.True(structs.IsNil(v))

	// nil pointer
	var p *int = nil
	is.True(structs.IsNil(p))

	// non-nil pointer
	x := 42
	p = &x
	is.False(structs.IsNil(p))

	// nil slice
	var s []int = nil
	is.True(structs.IsNil(s))

	// non-nil slice
	s = []int{1, 2, 3}
	is.False(structs.IsNil(s))

	// nil map
	var m map[string]int = nil
	is.True(structs.IsNil(m))

	// non-nil map
	m = map[string]int{"a": 1}
	is.False(structs.IsNil(m))

	// nil channel
	var ch chan int = nil
	is.True(structs.IsNil(ch))

	// non-nil channel
	ch = make(chan int)
	is.False(structs.IsNil(ch))

	// nil func
	var fn func() = nil
	is.True(structs.IsNil(fn))

	// non-nil func
	fn = func() {}
	is.False(structs.IsNil(fn))

	// nil interface value
	var i interface{} = nil
	is.True(structs.IsNil(i))

	// non-nil interface value
	i = 123
	is.False(structs.IsNil(i))

	// zero value (not nil)
	is.False(structs.IsNil(0))
	is.False(structs.IsNil(""))
	is.False(structs.IsNil(false))
}

type testStruct struct{}

func (testStruct) Foo()  {}
func (testStruct) Bar()  {}
func (*testStruct) Baz() {}
func (*testStruct) qux() {}

func TestGetMethodNames(t *testing.T) {
	ts := &testStruct{}
	methods, err := structs.GetMethodNames(ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]bool{"Foo": true, "Bar": true, "Baz": true}
	for _, m := range methods {
		if !want[m] {
			t.Errorf("unexpected method: %s", m)
		}
		delete(want, m)
	}
	for m := range want {
		t.Errorf("missing method: %s", m)
	}
}

func TestGetMethodNames_Nil(t *testing.T) {
	_, err := structs.GetMethodNames(nil)
	if err == nil {
		t.Error("expected error for nil value")
	}
}
