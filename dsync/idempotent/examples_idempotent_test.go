package idempotent_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/alextanhongpin/core/dsync/idempotent"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	redis "github.com/redis/go-redis/v9"
)

type Request struct {
	Age  int
	Name string
}

type Response struct {
	UserID int64
}

func ExampleMakeHandler() {
	ctx := context.Background()
	stop := redistest.Init()
	defer stop()

	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	client.FlushAll(ctx)

	defer client.Close()

	// Create a request object with the required fields.
	req := Request{
		Age:  10,
		Name: "john",
	}

	// Create a wait group to manage concurrent requests.
	var wg sync.WaitGroup
	wg.Add(3)

	// Start the first concurrent request.
	go func() {
		defer wg.Done()

		// Define the handler function that simulates the actual task
		fn := func(ctx context.Context, req Request) (*Response, error) {
			fmt.Printf("Executing get user #1: %+v\n", req)
			time.Sleep(40 * time.Millisecond)
			return &Response{UserID: 10}, nil
		}

		// Execute the idempotent operation and handle the response
		store := idempotent.NewRedisStore(client, nil)
		h := idempotent.MakeHandler(store, fn)
		v, shared, err := h.Do(ctx, "get-user", req)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Success #1: %+v %t\n", v, shared)
	}()

	// Start the second concurrent request.
	go func() {
		defer wg.Done()

		// Introduce a delay to simulate a second concurrent request.
		time.Sleep(25 * time.Millisecond)

		// Define the handler function that simulates the actual task.
		fn := func(ctx context.Context, req Request) (*Response, error) {
			fmt.Printf("Executing get user #2: %+v\n", req)
			return &Response{UserID: 10}, nil
		}

		// Execute the idempotent operation and handle the response.
		store := idempotent.NewRedisStore(client, nil)
		h := idempotent.MakeHandler(store, fn)
		_, _, err := h.Do(ctx, "get-user", req)
		if err == nil {
			fmt.Println(err)
			panic("want error, got nil")
		}

		// Check if the error is the expected ErrRequestInFlight.
		fmt.Println("Failed #2:", err)
		fmt.Println(errors.Is(err, idempotent.ErrRequestInFlight))
	}()

	// Start the third concurrent request.
	go func() {
		defer wg.Done()

		// Introduce a delay to simulate a third concurrent request.
		// This request happens after the first request completes.
		time.Sleep(60 * time.Millisecond)

		// Define the handler function that simulates the actual task.
		fn := func(ctx context.Context, req Request) (*Response, error) {
			fmt.Printf("Executing get user #3: %+v\n", req)
			return &Response{UserID: 10}, nil
		}

		// Execute the idempotent operation and handle the response.
		store := idempotent.NewRedisStore(client, nil)
		h := idempotent.MakeHandler(store, fn)
		v, shared, err := h.Do(ctx, "get-user", req)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Success #3: %+v %t\n", v, shared)
	}()

	wg.Wait()
	// Output:
	// Executing get user #1: {Age:10 Name:john}
	// Failed #2: idempotent: request in flight
	// true
	// Success #1: &{UserID:10} false
	// Success #3: &{UserID:10} true
}
