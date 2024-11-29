package rate_test

import (
	"errors"
	"testing"

	"github.com/alextanhongpin/core/sync/rate"
	"github.com/stretchr/testify/assert"
)

func TestLimit(t *testing.T) {
	badRequestErr := errors.New("bad request")

	t.Run("three consecutive errors", func(t *testing.T) {
		is := assert.New(t)
		limit := rate.NewLimiter(3)
		for range 3 {
			is.ErrorIs(limit.Do(func() error {
				return badRequestErr
			}), badRequestErr)
		}

		is.ErrorIs(limit.Do(func() error {
			return nil
		}), rate.ErrLimitExceeded)
		is.Equal(3, limit.Failure())
		is.Equal(0, limit.Success())
		is.Equal(3, limit.Total())
	})

	t.Run("two consecutive errors, one success, two consecutive errors", func(t *testing.T) {
		is := assert.New(t)
		limit := rate.NewLimiter(3)
		for range 2 {
			is.ErrorIs(limit.Do(func() error {
				return badRequestErr
			}), badRequestErr)
		}

		is.Nil(limit.Do(func() error {
			return nil
		}))

		for range 2 {
			is.ErrorIs(limit.Do(func() error {
				return badRequestErr
			}), badRequestErr)
		}

		is.ErrorIs(limit.Do(func() error {
			return nil
		}), rate.ErrLimitExceeded)

		is.Equal(4, limit.Failure())
		is.Equal(1, limit.Success())
		is.Equal(5, limit.Total())
	})

	t.Run("two consecutive errors, three successes, three consecutive errors", func(t *testing.T) {
		is := assert.New(t)
		limit := rate.NewLimiter(3)
		for range 2 {
			is.ErrorIs(limit.Do(func() error {
				return badRequestErr
			}), badRequestErr)
		}

		for range 3 {
			is.Nil(limit.Do(func() error {
				return nil
			}))
		}

		for range 3 {
			is.ErrorIs(limit.Do(func() error {
				return badRequestErr
			}), badRequestErr)
		}

		is.ErrorIs(limit.Do(func() error {
			return nil
		}), rate.ErrLimitExceeded)

		is.Equal(5, limit.Failure())
		is.Equal(3, limit.Success())
		is.Equal(8, limit.Total())
	})
}
