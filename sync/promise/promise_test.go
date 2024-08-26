package promise_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/promise"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()
var wantErr = errors.New("test: want error")

func TestStatus(t *testing.T) {
	t.Run("idle", func(t *testing.T) {
		p := promise.Deferred[int]()
		is := assert.New(t)
		is.Equal(promise.Idle, p.Status())
	})

	t.Run("fulfilled", func(t *testing.T) {
		p := promise.Deferred[int]()
		is := assert.New(t)
		is.Equal(promise.Idle, p.Status())
		n, err := p.Wait(func() (int, error) {
			is.Equal(promise.Pending, p.Status())
			return 42, nil
		})
		is.Nil(err)
		is.Equal(42, n)
		is.Equal(promise.Fulfilled, p.Status())
	})

	t.Run("rejected", func(t *testing.T) {
		p := promise.Deferred[int]()
		is := assert.New(t)
		is.Equal(promise.Idle, p.Status())
		n, err := p.Wait(func() (int, error) {
			is.Equal(promise.Pending, p.Status())
			return 0, wantErr
		})
		is.ErrorIs(err, wantErr)
		is.Equal(0, n)
		is.Equal(promise.Rejected, p.Status())
	})
}

func TestTimeout(t *testing.T) {
	t.Run("chain", func(t *testing.T) {
		p := promise.Deferred[int]().WithTimeout(0)
		time.Sleep(1 * time.Millisecond)

		_, err := p.Await()
		is := assert.New(t)
		is.Equal(promise.Rejected, p.Status())
		is.ErrorIs(err, promise.ErrTimeout)
	})

	t.Run("constructor", func(t *testing.T) {
		p := promise.WithTimeout[int](0)
		time.Sleep(1 * time.Millisecond)

		_, err := p.Await()
		is := assert.New(t)
		is.Equal(promise.Rejected, p.Status())
		is.ErrorIs(err, promise.ErrTimeout)
	})
}

func TestWait_Idempotent(t *testing.T) {
	is := assert.New(t)

	counter := new(atomic.Int64)
	p := promise.Deferred[int]()
	fn := func() (int, error) {
		counter.Add(1)
		return 42, nil
	}

	n := 10

	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()

			n, err := p.Wait(fn)
			is.Equal(42, n)
			is.Nil(err)
			is.Equal(promise.Fulfilled, p.Status())
		}()
	}

	wg.Wait()
	is.Equal(int64(1), counter.Load())
}
