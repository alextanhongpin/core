package structs_test

import (
	"errors"
	"testing"
	"time"

	"github.com/alextanhongpin/core/types/structs"
	"github.com/stretchr/testify/assert"
)

type Time struct {
	now time.Time
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
	var keyErr *structs.KeyError
	is.True(errors.As(err, &keyErr))
	is.Equal(keyErr.Path, "structs_test.User.Name")
	is.Equal(keyErr.Key, "Name")

	u.Name = "John Appleseed"
	err = structs.NonZero(u)
	is.Nil(err)
}
