package idempotency_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/idempotency"
	"github.com/stretchr/testify/assert"
)

func TestRequestReplyInMemory(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	store := idempotency.NewInMemoryStore[*Response]()

	do := func() {
		handler := idempotency.NewRequestReply(store, idempotency.RequestReplyOption[Request, *Response]{
			LockTimeout:     5 * time.Second, // Default is 1 minute.
			RetentionPeriod: 1 * time.Minute, // Default is 24 hour.
			Handler: idempotency.RequestReplyHandler[Request, *Response](func(ctx context.Context, req Request) (*Response, error) {
				// Simulate critical section.
				time.Sleep(100 * time.Millisecond)

				return &Response{
					Name: "replied:" + req.Name,
				}, nil
			}),
		})

		res, err := handler.Exec(ctx, SomeOperationKey.Format("xyz"), Request{
			ID:   "payout-123",
			Name: "foo",
		})
		if err != nil {
			assert.ErrorIs(err, idempotency.ErrRequestInFlight, err)
		} else {
			assert.Equal("replied:foo", res.Name)
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	race := make(chan bool)

	go func() {
		defer wg.Done()

		<-race
		do()
	}()

	go func() {
		defer wg.Done()

		<-race
		do()
	}()

	time.Sleep(100 * time.Millisecond)

	// Run concurrently.
	close(race)

	wg.Wait()

	res, err := store.Load(ctx, "some-operation:xyz")
	assert.Nil(err)

	assert.Equal("success", string(res.Status))
	assert.Equal("w93v/T90sFbhkHDcVqEfX1HWwArDAIFdNnppRNwjuKg=", res.Request)
	assert.Equal("replied:foo", res.Response.Name)
}

func TestRequestInMemory(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	store := idempotency.NewInMemoryStore[any]()

	do := func() {
		handler := idempotency.NewRequest(store, idempotency.RequestOption[Request]{
			LockTimeout:     5 * time.Second, // Default is 1 minute.
			RetentionPeriod: 1 * time.Minute, // Default is 24 hour.
			Handler: idempotency.RequestHandler[Request](func(ctx context.Context, req Request) error {
				// Simulate critical section.
				time.Sleep(100 * time.Millisecond)

				return nil
			}),
		})

		err := handler.Exec(ctx, SomeOperationKey.Format("xyz"), Request{
			ID:   "payout-123",
			Name: "foo",
		})
		if err != nil {
			assert.ErrorIs(err, idempotency.ErrRequestInFlight, err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	race := make(chan bool)

	go func() {
		defer wg.Done()

		<-race
		do()
	}()

	go func() {
		defer wg.Done()

		<-race
		do()
	}()

	time.Sleep(100 * time.Millisecond)

	// Run concurrently.
	close(race)

	wg.Wait()

	res, err := store.Load(ctx, "some-operation:xyz")
	assert.Nil(err)

	assert.Equal("success", string(res.Status))
	assert.Equal("w93v/T90sFbhkHDcVqEfX1HWwArDAIFdNnppRNwjuKg=", res.Request)
	assert.Nil(res.Response)
}
