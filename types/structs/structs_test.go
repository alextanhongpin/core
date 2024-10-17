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
