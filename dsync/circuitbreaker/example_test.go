package circuitbreaker_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	redis "github.com/redis/go-redis/v9"
)

// The examples run against the Redis container started in TestMain.
// In your code, use your own address, e.g. "localhost:6379".
func newExampleClient(keys ...string) *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: redistest.Addr()})
	ctx := context.Background()
	if err := circuitbreaker.Setup(ctx, client); err != nil {
		panic(err)
	}
	client.Del(ctx, keys...) // Start from a clean state.
	return client
}

func Example() {
	ctx := context.Background()
	client := newExampleClient("cb:example:inventory")
	defer client.Close()

	cb, err := circuitbreaker.New(client, circuitbreaker.Config{
		FailureThreshold: 3,                // open after 3 failures ...
		FailurePeriod:    10 * time.Second, // ... with no more than 10s between them
		OpenTimeout:      30 * time.Second, // then reject calls for 30s
	})
	if err != nil {
		panic(err)
	}

	errUnavailable := errors.New("inventory service unavailable")
	for i := 1; i <= 4; i++ {
		err := cb.Do(ctx, "cb:example:inventory", func() error {
			return errUnavailable // The real outbound call goes here.
		})
		switch {
		case errors.Is(err, circuitbreaker.ErrOpened):
			fmt.Printf("call %d: rejected, serving cached inventory\n", i)
		case err != nil:
			fmt.Printf("call %d: failed: %v\n", i, err)
		}
	}

	status, err := cb.Status(ctx, "cb:example:inventory")
	if err != nil {
		panic(err)
	}
	fmt.Println("status:", status)
	// Output:
	// call 1: failed: inventory service unavailable
	// call 2: failed: inventory service unavailable
	// call 3: failed: inventory service unavailable
	// call 4: rejected, serving cached inventory
	// status: opened
}

// Wrap an outbound HTTP call with Func so callers keep the original signature.
func ExampleFunc() {
	client := newExampleClient("cb:example:users-api")
	defer client.Close()

	// A downstream API that is down.
	var hits int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer api.Close()

	getUser := func(ctx context.Context, id string) (string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, api.URL+"/users/"+id, nil)
		if err != nil {
			return "", err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 500 {
			return "", fmt.Errorf("users api: %s", resp.Status)
		}
		return "user " + id, nil
	}

	cb := circuitbreaker.MustNew(client, circuitbreaker.Config{FailureThreshold: 2})
	getUser = circuitbreaker.Func(getUser, cb, func(ctx context.Context, id string) (string, error) {
		return "cb:example:users-api", nil // One breaker for the whole dependency.
	})

	for range 4 {
		_, err := getUser(context.Background(), "42")
		fmt.Println(err)
	}
	fmt.Println("requests that reached the API:", hits)
	// Output:
	// users api: 502 Bad Gateway
	// users api: 502 Bad Gateway
	// cb: opened
	// cb: opened
	// requests that reached the API: 2
}

// Operators can force a breaker open during an incident and close it again.
func ExampleCircuitBreaker_SetStatus() {
	ctx := context.Background()
	client := newExampleClient("cb:example:payments")
	defer client.Close()

	cb := circuitbreaker.MustNew(client, circuitbreaker.Config{})
	if err := cb.SetStatus(ctx, "cb:example:payments", circuitbreaker.ForcedOpen); err != nil {
		panic(err)
	}
	err := cb.Do(ctx, "cb:example:payments", func() error { return nil })
	fmt.Println(errors.Is(err, circuitbreaker.ErrOpened))

	if err := cb.SetStatus(ctx, "cb:example:payments", circuitbreaker.Closed); err != nil {
		panic(err)
	}
	err = cb.Do(ctx, "cb:example:payments", func() error { return nil })
	fmt.Println(err)
	// Output:
	// true
	// <nil>
}
