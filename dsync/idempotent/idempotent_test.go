package idempotent_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/alextanhongpin/core/dsync/idempotent"
	"github.com/alextanhongpin/core/storage/redis/redistest"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	defer stop()

	m.Run()
}

func TestRedisStore(t *testing.T) {
	fn := func(ctx context.Context, req []byte) ([]byte, error) {
		return []byte("world"), nil
	}

	store := idempotent.NewRedisStore(redistest.Client(t))
	res, shared, err := store.Do(ctx, t.Name(), fn, []byte("hello"), time.Minute, time.Hour)
	is := assert.New(t)
	is.Nil(err)
	is.False(shared)
	is.Equal([]byte("world"), res)

	res, shared, err = store.Do(ctx, t.Name(), fn, []byte("hello"), time.Minute, time.Hour)
	is.Nil(err)
	is.True(shared)
	is.Equal([]byte("world"), res)
}

func TestMakeHandler(t *testing.T) {
	fn := func(ctx context.Context, req string) (string, error) {
		return "world", nil
	}
	h := idempotent.NewHandler(redistest.Client(t), fn)

	res, shared, err := h.Handle(ctx, t.Name(), "hello", time.Minute, time.Hour)
	is := assert.New(t)
	is.Nil(err)
	is.False(shared)
	is.Equal("world", res)

	res, shared, err = h.Handle(ctx, t.Name(), "hello", time.Minute, time.Hour)
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

	fn := func(ctx context.Context, req Request) (*Response, error) {
		invoked.Add(1)
		time.Sleep(100 * time.Millisecond)

		return &Response{
			Msg: strings.ToUpper(req.Msg),
		}, nil
	}

	client := redistest.Client(t)
	h := idempotent.NewHandler(client, fn)
	n := 10

	is := assert.New(t)

	var wg sync.WaitGroup
	wg.Add(3 * n)

	for range n {
		go func() {
			defer wg.Done()

			res, shared, err := h.Handle(ctx, t.Name(), Request{Msg: "hello"}, time.Minute, time.Hour)
			is.Equal("HELLO", res.Msg)
			is.Nil(err)
			if shared {
				counter.Add(1)
			}
		}()

		go func() {
			defer wg.Done()

			time.Sleep(50 * time.Millisecond)
			h := idempotent.NewHandler(client, fn)
			res, shared, err := h.Handle(ctx, t.Name(), Request{Msg: "hello"}, time.Minute, time.Hour)
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

		go func() {
			defer wg.Done()
			time.Sleep(150 * time.Millisecond)

			h := idempotent.NewHandler(client, fn)
			res, shared, err := h.Handle(ctx, t.Name(), Request{Msg: "hello"}, time.Minute, time.Hour)
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
	is.Equal(int64(1), invoked.Load())
	is.Equal(int64(n*2-1), counter.Load())
	is.Equal(int64(n), inFlight.Load())
}

// TestExtendLock test the scenario where the callback function takes a longer
// time than the lock expiry, and the lock expired before the callback function
// completes.
// We expect the lock to be refresh periodically.
func TestExtendLock(t *testing.T) {
	client := redistest.Client(t)

	fn := func(ctx context.Context, req string) (int, error) {
		// slow function
		time.Sleep(250 * time.Millisecond)
		return 42, nil
	}

	h := idempotent.NewHandler(client, fn)
	_, _, err := h.Handle(ctx, t.Name(), "world", 100*time.Millisecond, 200*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
}
