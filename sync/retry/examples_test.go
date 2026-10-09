package retry_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRetry_Do() {
	r, err := retry.New(retry.Config{MaxRetries: 2, Backoff: retry.NewConstantBackoff(0), Throttler: retry.NewNoopThrottler()})
	if err != nil {
		panic(err)
	}
	calls := 0
	err = r.Do(context.Background(), func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("temporary")
		}
		return nil
	})
	fmt.Println(calls, err)
	// Output: 3 <nil>
}

func ExampleFunc() {
	r, err := retry.New(retry.Config{})
	if err != nil {
		panic(err)
	}
	lookup := retry.Func(func(ctx context.Context, id string) (string, error) { return "user " + id, nil }, r)
	user, err := lookup(context.Background(), "123")
	fmt.Println(user, err)
	// Output: user 123 <nil>
}

func ExampleNewRoundTripper() {
	r, err := retry.New(retry.Config{MaxRetries: 1, Backoff: retry.NewConstantBackoff(0), Throttler: retry.NewNoopThrottler()})
	if err != nil {
		panic(err)
	}
	calls := 0
	transport := retry.NewRoundTripper(transportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return response(http.StatusServiceUnavailable, http.NoBody), nil
	}), r)
	client := &http.Client{Transport: transport}
	resp, err := client.Get("http://example.test")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp.StatusCode, calls)
	// Output: 503 2
}
