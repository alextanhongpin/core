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

	var u User
	err := structs.NonZero(u)

	is := assert.New(t)
	var fieldErr *structs.FieldError
	is.True(errors.As(err, &fieldErr))
	is.Equal(fieldErr.Path, "structs_test.User.Name")
	is.Equal(fieldErr.Field, "Name")

	u.Name = "John Appleseed"
	err = structs.NonZero(u)
	is.Nil(err)
}
