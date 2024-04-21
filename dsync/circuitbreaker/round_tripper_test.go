package circuitbreaker_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"

	"github.com/alextanhongpin/core/dsync/circuitbreaker"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/redis/go-redis/v9"
)

func ExampleRoundTripper() {
	client := redis.NewClient(&redis.Options{
		Addr: redistest.Addr(),
	})
	defer func() {
		client.FlushAll(ctx).Err()
		client.Close()
	}()

	opt := circuitbreaker.NewOption()
	opt.FailureThreshold = 3

	cb := circuitbreaker.New(client, opt)
	cb.OnStateChanged = func(ctx context.Context, from, to circuitbreaker.Status) {
		fmt.Printf("status changed from %s to %s\n", from, to)
	}

	status := http.StatusBadRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	defer ts.Close()

	httpClient := ts.Client()
	httpClient.Transport = circuitbreaker.NewRoundTripper(httpClient.Transport, cb)

	re := regexp.MustCompile(`\d{5}`)

	// Ignores http status 4xx
	for i := 0; i < int(opt.FailureThreshold); i++ {
		_, err := httpClient.Get(ts.URL)
		if err != nil {
			// Replace port since it changes dynamically and breaks the test.
			msg := re.ReplaceAllString(err.Error(), "8080")
			fmt.Println(msg)
			continue
		}
	}

	// Handles http status 5xx
	status = http.StatusInternalServerError

	for i := 0; i < int(opt.FailureThreshold)+1; i++ {
		_, err := httpClient.Get(ts.URL)
		if err != nil {
			// Replace port since it changes dynamically and breaks the test.
			msg := re.ReplaceAllString(err.Error(), "8080")
			fmt.Println(msg)
			continue
		}
	}

	// Output
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// status changed from closed to open
	// Get "http://127.0.0.1:8080": circuit-breaker: broken
}
