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

func TestMain(m *testing.M) {
	stop := redistest.Init()
	code := m.Run()
	stop()
	os.Exit(code)
}

func TestCircuitBreaker_Do_RedisStore(t *testing.T) {
	now := time.Now()

	var statuses []circuitbreaker.Status
	// Create a new circuit breaker with default options.
	opt := circuitbreaker.NewOption()
	opt.SuccessThreshold = 3
	opt.FailureThreshold = 3
	opt.BreakDuration = 5 * time.Second
	opt.OnStateChanged = func(ctx context.Context, from, to circuitbreaker.Status) {
		statuses = append(statuses, from, to)
	}
	opt.Store = circuitbreaker.NewRedisStore(setupRedis(t))
	opt.Now = func() time.Time {
		return now
	}

	// Create a new circuit breaker.
	cb := circuitbreaker.New(opt)

	key := t.Name()

	run := func(t *testing.T, n int, wantErr, gotErr error) {
		t.Helper()

		for i := 0; i < n; i++ {
			err := cb.Do(ctx, key, func() error {
				return wantErr
			})
			assert.ErrorIs(t, err, gotErr)
		}
		v, err := opt.Store.Get(ctx, key)
		if err != nil {
			assert.Nil(t, err)
		}
		t.Logf("%+v\n", v)
	}

	testStatus := func(t *testing.T, want circuitbreaker.Status) {
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
		testStatus(t, circuitbreaker.Closed)
	})

	t.Run("status changed to open", func(t *testing.T) {
		var wantErr = errors.New("wantErr")
		run(t, opt.FailureThreshold, wantErr, wantErr)
		run(t, 1, wantErr, circuitbreaker.ErrBrokenCircuit)
		testStatus(t, circuitbreaker.Open)
		testStatusChanged(t, circuitbreaker.Closed, circuitbreaker.Open)
	})

	t.Run("status changed to half-open", func(t *testing.T) {
		now = now.Add(opt.BreakDuration)

		run(t, 1, nil, nil)
		testStatus(t, circuitbreaker.HalfOpen)
		testStatusChanged(t, circuitbreaker.Open, circuitbreaker.HalfOpen)
	})

	t.Run("status changed to closed", func(t *testing.T) {
		run(t, opt.SuccessThreshold, nil, nil)
		testStatus(t, circuitbreaker.Closed)
		testStatusChanged(t, circuitbreaker.HalfOpen, circuitbreaker.Closed)
	})
}

func TestCircuitBreaker_Do_RedisStore_ConcurrentWrite(t *testing.T) {
	var statuses []circuitbreaker.Status
	// Create a new circuit breaker with default options.
	opt := circuitbreaker.NewOption()
	opt.SuccessThreshold = 3
	opt.FailureThreshold = 5
	opt.BreakDuration = 5 * time.Second
	opt.OnStateChanged = func(ctx context.Context, from, to circuitbreaker.Status) {
		statuses = append(statuses, from, to)
	}
	opt.Store = circuitbreaker.NewRedisStore(setupRedis(t))

	// Create a new circuit breaker.
	cb := circuitbreaker.New(opt)

	key := t.Name()

	var wantErr = errors.New("wantErr")
	n := 5

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
	for err := range errs {
		if err != nil && !errors.Is(err, wantErr) {
			t.Fatal(err)
		}
	}
	res, err := opt.Store.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%+v\n", res)
	t.Log(statuses)
}

func TestCircuitBreaker_Do_RedisStore_MultipleInstance(t *testing.T) {
	opt := circuitbreaker.NewOption()
	opt.SuccessThreshold = 3
	opt.FailureThreshold = 5
	opt.BreakDuration = 5 * time.Second
	opt.Store = circuitbreaker.NewRedisStore(setupRedis(t))

	key := t.Name()
	fn := func() error {
		return errors.New("want err")
	}

	{
		cb := circuitbreaker.New(opt)
		for i := 0; i < opt.FailureThreshold; i++ {
			err := cb.Do(ctx, key, fn)
			if err == nil {
				t.Fatal("want err, got nil")
			}
		}
	}

	{
		cb := circuitbreaker.New(opt)
		err := cb.Do(ctx, key, fn)
		if !errors.Is(err, circuitbreaker.ErrBrokenCircuit) {
			t.Fatalf("want %v, got %v", circuitbreaker.ErrBrokenCircuit, err)
		}
	}
}

func setupRedis(t *testing.T) *redis.Client {
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
