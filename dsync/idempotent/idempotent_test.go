package idempotent_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/alextanhongpin/core/dsync/idempotent"
	"github.com/alextanhongpin/dbtx/testing/redistest"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	defer stop()

	m.Run()
}

func TestHandlerFunc(t *testing.T) {
	client := redistest.Client(t)
	t.Cleanup(func() {
		scanAll(client)
	})
	fn := func(ctx context.Context, req string) (string, error) {
		scanAll(client)
		return "world", nil
	}
	idb := idempotent.MustNewWithRedis(client)
	idp := idb.MustHandlerFunc(fn, idempotent.HandlerConfig{})

	res, shared, err := idp.Do(ctx, t.Name(), "hello")
	is := assert.New(t)
	is.Nil(err)
	is.False(shared)
	is.Equal("world", res)

	res, shared, err = idp.Do(ctx, t.Name(), "hello")
	is.Nil(err)
	is.True(shared)
	is.Equal("world", res)
}

func TestConcurrent(t *testing.T) {
	type Request struct {
		Msg string
	}
	type Response struct {
		Msg string
	}

	invoked := new(atomic.Int64)
	counter := new(atomic.Int64)
	inFlight := new(atomic.Int64)

	const fnDuration = 100 * time.Millisecond

	fn := func(ctx context.Context, req Request) (*Response, error) {
		invoked.Add(1)
		time.Sleep(fnDuration)

		return &Response{
			Msg: strings.ToUpper(req.Msg),
		}, nil
	}

	client := redistest.Client(t)
	t.Cleanup(func() {
		scanAll(client)
	})
	idb := idempotent.MustNewWithRedis(client)
	idp := idb.MustHandlerFunc(fn, idempotent.HandlerConfig{})
	n := 10

	is := assert.New(t)

	var wg sync.WaitGroup
	wg.Add(3 * n)

	for range n {
		// Group 1: arrive immediately; one executes, the rest (n-1) block on the
		// in-process mutex and receive the cached result once the lock is released.
		go func() {
			defer wg.Done()

			res, shared, err := idp.Do(ctx, t.Name(), Request{Msg: "hello"})
			is.Equal("HELLO", res.Msg)
			is.Nil(err)
			if shared {
				counter.Add(1)
			}
		}()

		// Group 2: arrive mid-flight (before fn completes).
		// Simulates cross-process callers (no in-process lock contention) so each sees the
		// in-flight token and receives ErrRequestInFlight immediately.
		go func() {
			defer wg.Done()

			time.Sleep(50 * time.Millisecond)
			idp := idempotent.MustNewWithRedis(client).MustHandlerFunc(fn, idempotent.HandlerConfig{})
			res, shared, err := idp.Do(ctx, t.Name(), Request{Msg: "hello"})
			if errors.Is(err, idempotent.ErrRequestInFlight) {
				inFlight.Add(1)
				return
			}
			is.Equal("HELLO", res.Msg)
			is.Nil(err)
			if shared {
				counter.Add(1)
			}
		}()

		// Group 3: arrive well after fn completes (300ms >> fnDuration + overhead).
		// Simulates cross-process callers arriving after completion and receiving the cached result.
		go func() {
			defer wg.Done()
			time.Sleep(300 * time.Millisecond)

			idp := idempotent.MustNewWithRedis(client).MustHandlerFunc(fn, idempotent.HandlerConfig{})
			res, shared, err := idp.Do(ctx, t.Name(), Request{Msg: "hello"})
			if errors.Is(err, idempotent.ErrRequestInFlight) {
				inFlight.Add(1)
				return
			}
			is.Equal("HELLO", res.Msg)
			is.Nil(err)
			if shared {
				counter.Add(1)
			}
		}()
	}

	wg.Wait()
	// fn must be invoked exactly once.
	is.Equal(int64(1), invoked.Load())
	// Group 1 produces (n-1) shared hits; Group 3 produces n shared hits.
	is.Equal(int64(n*2-1), counter.Load())
	// All Group 2 goroutines see ErrRequestInFlight.
	is.Equal(int64(n), inFlight.Load())
}

// TestExtendLock tests the scenario where the callback function takes longer
// than the lock expiry, verifying that the lock TTL is refreshed periodically.
func TestExtendLock(t *testing.T) {
	client := redistest.Client(t)
	t.Cleanup(func() {
		scanAll(client)
	})

	fn := func(ctx context.Context, req string) (int, error) {
		time.Sleep(250 * time.Millisecond)
		return 42, nil
	}

	idb := idempotent.MustNewWithRedis(client)
	idp := idb.MustHandlerFunc(fn, idempotent.HandlerConfig{
		LockTTL: 100 * time.Millisecond,
		KeepTTL: 200 * time.Millisecond,
	})
	_, _, err := idp.Do(ctx, t.Name(), "world")
	if err != nil {
		t.Fatal(err)
	}
}

func TestEmptyKey(t *testing.T) {
	client := redistest.Client(t)
	idb := idempotent.MustNewWithRedis(client)
	idp := idb.MustHandlerFunc(func(ctx context.Context, req string) (string, error) {
		return "ok", nil
	}, idempotent.HandlerConfig{})

	_, _, err := idp.Do(ctx, "", "hello")
	assert.ErrorIs(t, err, idempotent.ErrEmptyKey)
}

func TestPanicRecovery(t *testing.T) {
	client := redistest.Client(t)
	idb := idempotent.MustNewWithRedis(client)
	var attempts int
	idp := idb.MustHandlerFunc(func(ctx context.Context, req string) (string, error) {
		attempts++
		if attempts == 1 {
			panic("something went horribly wrong")
		}
		return "recovered", nil
	}, idempotent.HandlerConfig{})

	// First call should panic on the caller's goroutine, and release the lock in Redis.
	assert.Panics(t, func() {
		_, _, _ = idp.Do(ctx, t.Name(), "payload")
	})

	// Subsequent call should be able to acquire the lock and succeed.
	res, shared, err := idp.Do(ctx, t.Name(), "payload")
	assert.NoError(t, err)
	assert.False(t, shared)
	assert.Equal(t, "recovered", res)
}

func TestUnexportedFields(t *testing.T) {
	type RequestWithPrivate struct {
		Public  string
		private int
	}
	type ResponseWithPrivate struct {
		Result  string
		private int
	}

	client := redistest.Client(t)
	idb := idempotent.MustNewWithRedis(client)
	idp := idb.MustHandlerFunc(func(ctx context.Context, req RequestWithPrivate) (ResponseWithPrivate, error) {
		return ResponseWithPrivate{Result: req.Public + "-done", private: 42}, nil
	}, idempotent.HandlerConfig{})

	req := RequestWithPrivate{Public: "test", private: 1}
	res1, shared1, err := idp.Do(ctx, t.Name(), req)
	assert.NoError(t, err)
	assert.False(t, shared1)
	assert.Equal(t, "test-done", res1.Result)

	// Second call should return cached response without panicking on unexported fields.
	res2, shared2, err := idp.Do(ctx, t.Name(), req)
	assert.NoError(t, err)
	assert.True(t, shared2)
	assert.Equal(t, "test-done", res2.Result)
}

func TestRequestMismatch(t *testing.T) {
	client := redistest.Client(t)
	idb := idempotent.MustNewWithRedis(client)
	idp := idb.MustHandlerFunc(func(ctx context.Context, req string) (string, error) {
		return "ok", nil
	}, idempotent.HandlerConfig{})

	res, shared, err := idp.Do(ctx, t.Name(), "first")
	assert.NoError(t, err)
	assert.False(t, shared)
	assert.Equal(t, "ok", res)

	// Same key, different request payload.
	_, _, err = idp.Do(ctx, t.Name(), "second")
	assert.ErrorIs(t, err, idempotent.ErrRequestMismatch)
}

func TestZeroConfig(t *testing.T) {
	client := redistest.Client(t)
	idb := idempotent.MustNewWithRedis(client)
	// Zero-valued config should not panic with time.NewTicker(0).
	idp := idb.MustHandlerFunc(func(ctx context.Context, req string) (string, error) {
		return "ok", nil
	}, idempotent.HandlerConfig{})

	res, shared, err := idp.Do(ctx, t.Name(), "test")
	assert.NoError(t, err)
	assert.False(t, shared)
	assert.Equal(t, "ok", res)
}

func TestContextCancellation(t *testing.T) {
	client := redistest.Client(t)
	idb := idempotent.MustNewWithRedis(client)

	var once sync.Once
	started := make(chan struct{})
	idp := idb.MustHandlerFunc(func(ctx context.Context, req string) (string, error) {
		once.Do(func() { close(started) })
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return "done", nil
		}
	}, idempotent.HandlerConfig{})

	ctxCancel, cancel := context.WithCancel(ctx)
	errCh := make(chan error, 1)

	go func() {
		_, _, err := idp.Do(ctxCancel, t.Name(), "hello")
		errCh <- err
	}()

	<-started
	cancel()

	err := <-errCh
	assert.ErrorIs(t, err, context.Canceled)

	// Once the worker finishes cleanup, subsequent request can run.
	assert.Eventually(t, func() bool {
		_, _, err := idp.Do(ctx, t.Name(), "hello")
		return err == nil
	}, 1*time.Second, 50*time.Millisecond)
}

func scanAll(rdb *redis.Client) {
	// Use Scan to safely find all keys without blocking Redis.
	iter := rdb.Scan(ctx, 0, "*", 0).Iterator()

	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		log.Fatalf("Failed to scan keys: %v", err)
	}

	if len(keys) == 0 {
		fmt.Println("No keys found in Redis.")
		return
	}

	// Fetch all values efficiently using MGet (Multi-Get).
	values, err := rdb.MGet(ctx, keys...).Result()
	if err != nil {
		log.Fatalf("Failed to MGet values: %v", err)
	}

	// Map keys to their respective values.
	// Note: MGet returns nil for keys that don't exist or expired during execution.
	fmt.Println("--- Key-Value Pairs ---")
	for i, key := range keys {
		val := values[i]
		if val == nil {
			fmt.Printf("%s: <nil> (key may have expired)\n", key)
		} else {
			fmt.Printf("%s: %v\n", key, val)
		}
	}
}
