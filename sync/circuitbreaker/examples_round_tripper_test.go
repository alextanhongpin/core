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
	opt.OnStateChanged = func(from, to circuitbreaker.Status) {
		fmt.Printf("status changed from %s to %s\n", from, to)
	}
	cb := circuitbreaker.New(opt)
	fmt.Println("initial status:", cb.Status())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := ts.Client()
	client.Transport = &circuitbreaker.RoundTripper{
		Transport:      client.Transport,
		CircuitBreaker: cb,
	}

	re := regexp.MustCompile(`\d{5}`)
	// Opens after failure ratio exceeded.
	for i := 0; i <= int(opt.FailureThreshold+1); i++ {
		resp, err := client.Get(ts.URL)
		if err != nil {
			// Replace port since it changes dynamically and breaks the test.
			msg := re.ReplaceAllString(err.Error(), "8080")
			fmt.Println(msg)
		}
		if resp == nil {
			continue
		}
		defer resp.Body.Close()
	}

	// Output:
	// initial status: closed
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
	// status changed from closed to open
	// Get "http://127.0.0.1:8080": circuit-breaker: broken
}
