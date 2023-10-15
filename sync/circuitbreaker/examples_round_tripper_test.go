package circuitbreaker_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
)

func ExampleRoundTripper() {
	opt := circuitbreaker.NewOption()
	opt.BreakDuration = 100 * time.Millisecond
	opt.SamplingDuration = 1 * time.Second
	cb := circuitbreaker.New[any](opt)
	cb.OnStateChanged = func(from, to circuitbreaker.Status) {
		fmt.Printf("status changed from %s to %s\n", from, to)
	}
	fmt.Println("initial status:", cb.Status())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := ts.Client()
	client.Transport = circuitbreaker.NewRoundTripper(client.Transport)

	re := regexp.MustCompile(`\d{5}`)
	// Opens after failure ratio exceeded.
	for i := 0; i < int(opt.FailureThreshold)+1; i++ {
		_, err := client.Get(ts.URL)
		if err != nil {
			// Replace port since it changes dynamically and breaks the test.
			msg := re.ReplaceAllString(err.Error(), "8080")
			fmt.Println(msg)
			continue
		}
	}

	// Output:
	// initial status: closed
	// Get "http://127.0.0.1:8080": circuit-breaker: failing: 500 Internal Server Error
	// Get "http://127.0.0.1:8080": circuit-breaker: failing: 500 Internal Server Error
	// Get "http://127.0.0.1:8080": circuit-breaker: failing: 500 Internal Server Error
	// Get "http://127.0.0.1:8080": circuit-breaker: failing: 500 Internal Server Error
	// Get "http://127.0.0.1:8080": circuit-breaker: failing: 500 Internal Server Error
	// Get "http://127.0.0.1:8080": circuit-breaker: failing: 500 Internal Server Error
	// Get "http://127.0.0.1:8080": circuit-breaker: failing: 500 Internal Server Error
	// Get "http://127.0.0.1:8080": circuit-breaker: failing: 500 Internal Server Error
	// Get "http://127.0.0.1:8080": circuit-breaker: failing: 500 Internal Server Error
	// Get "http://127.0.0.1:8080": circuit-breaker: failing: 500 Internal Server Error
	// Get "http://127.0.0.1:8080": circuit-breaker: broken
}
