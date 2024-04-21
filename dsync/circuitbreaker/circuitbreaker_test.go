package circuitbreaker_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestCircuitBreaker_Do(t *testing.T) {

	// Create a new circuit breaker with default options.
	opt := circuitbreaker.NewOption()

	// Create a new circuit breaker.
	cb := circuitbreaker.New(newClient(t), opt)

	// Record status changes.
	var statuses []circuitbreaker.Status
	cb.OnStateChanged = func(ctx context.Context, from, to circuitbreaker.Status) {
		statuses = append(statuses, from, to)
	}

	now := time.Now()
	cb.Now = func() time.Time {
		return now
	}

	key := t.Name()

	// run executes the circuitbreaker n times, and checks if the error matches.
	run := func(t *testing.T, n int, wantErr, gotErr error) {
		t.Helper()

		is := assert.New(t)
		for i := 0; i < n; i++ {
			err := cb.Do(ctx, key, func() error {
				return wantErr
			})
			is.ErrorIs(err, gotErr)
		}
	}

	testStatusEqual := func(t *testing.T, want circuitbreaker.Status) {
		t.Helper()

		is := assert.New(t)
		status, err := cb.Status(ctx, key)
		is.Nil(err)
		is.Equal(want, status)
	}

	testStatusChanged := func(t *testing.T, from, to circuitbreaker.Status) {
		t.Helper()

		is := assert.New(t)
		i := len(statuses) - 2
		is.Equal(from, statuses[i])
		is.Equal(to, statuses[i+1])
	}

	t.Run("initial status is closed", func(t *testing.T) {
		testStatusEqual(t, circuitbreaker.Closed)
	})

	t.Run("status changed to open", func(t *testing.T) {
		var wantErr = errors.New("wantErr")
		run(t, opt.FailureThreshold, wantErr, wantErr)
		run(t, 1, wantErr, circuitbreaker.ErrBrokenCircuit)
		testStatusEqual(t, circuitbreaker.Open)
		testStatusChanged(t, circuitbreaker.Closed, circuitbreaker.Open)
	})

	t.Run("status changed to half-open", func(t *testing.T) {
		now = now.Add(opt.BreakDuration)

		run(t, 1, nil, nil)
		testStatusEqual(t, circuitbreaker.HalfOpen)
		testStatusChanged(t, circuitbreaker.Open, circuitbreaker.HalfOpen)
	})

	t.Run("status changed to closed", func(t *testing.T) {
		run(t, opt.SuccessThreshold, nil, nil)
		testStatusEqual(t, circuitbreaker.Closed)
		testStatusChanged(t, circuitbreaker.HalfOpen, circuitbreaker.Closed)
	})

	t.Run("status changed to open", func(t *testing.T) {
		var wantErr = errors.New("wantErr")
		run(t, opt.FailureThreshold, wantErr, wantErr)
		run(t, 1, wantErr, circuitbreaker.ErrBrokenCircuit)
		testStatusEqual(t, circuitbreaker.Open)
		testStatusChanged(t, circuitbreaker.Closed, circuitbreaker.Open)
	})

	t.Run("status changed to half-open", func(t *testing.T) {
		now = now.Add(opt.BreakDuration)

		run(t, 1, nil, nil)
		testStatusEqual(t, circuitbreaker.HalfOpen)
		testStatusChanged(t, circuitbreaker.Open, circuitbreaker.HalfOpen)
	})

	t.Run("status changed to open", func(t *testing.T) {
		var wantErr = errors.New("want error")
		run(t, 1, wantErr, wantErr)
		testStatusEqual(t, circuitbreaker.Open)
		testStatusChanged(t, circuitbreaker.HalfOpen, circuitbreaker.Open)
	})
}

func TestCircuitBreaker_Do_ConcurrentWrite(t *testing.T) {
	// Create a new circuit breaker with default options.
	opt := circuitbreaker.NewOption()

	// Create a new circuit breaker.
	cb := circuitbreaker.New(newClient(t), opt)

	var statuses []circuitbreaker.Status
	cb.OnStateChanged = func(ctx context.Context, from, to circuitbreaker.Status) {
		statuses = append(statuses, from, to)
	}

	key := t.Name()

	var wantErr = errors.New("wantErr")
	n := opt.FailureThreshold + 1

	var wg sync.WaitGroup
	wg.Add(n)
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			errs <- cb.Do(ctx, key, func() error {
				time.Sleep(time.Duration(i) * 100 * time.Millisecond)
				return wantErr
			})
		}()
	}

	wg.Wait()
	close(errs)

	is := assert.New(t)
	for err := range errs {
		is.ErrorIs(err, wantErr)
	}

	res, err := cb.Status(ctx, key)
	is.Nil(err)
	is.Equal(circuitbreaker.Open, res)
	is.Equal([]circuitbreaker.Status{circuitbreaker.Closed, circuitbreaker.Open}, statuses)
}

func TestCircuitBreaker_Do_MultipleInstance(t *testing.T) {
	var wantErr = errors.New("want error")
	key := t.Name()
	fn := func() error {
		return wantErr
	}

	is := assert.New(t)
	{
		opt := circuitbreaker.NewOption()
		cb := circuitbreaker.New(newClient(t), opt)
		for i := 0; i < opt.FailureThreshold; i++ {
			err := cb.Do(ctx, key, fn)
			is.ErrorIs(err, wantErr)
		}
	}

	{
		opt := circuitbreaker.NewOption()
		cb := circuitbreaker.New(newClient(t), opt)
		err := cb.Do(ctx, key, fn)
		is.ErrorIs(err, circuitbreaker.ErrBrokenCircuit)
	}
}

func newClient(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})

	t.Helper()
	t.Cleanup(func() {
		client.FlushAll(ctx).Err()
		client.Close()
	})

	return client
}
