package circuitbreaker_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
)

func ExampleTransporter() {
	cb := circuitbreaker.New()
	cb.BreakDuration = 100 * time.Millisecond
	cb.SamplingDuration = 1 * time.Second

	fmt.Println("initial status:")
	fmt.Println(cb.Status())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := ts.Client()
	client.Transport = circuitbreaker.NewTransporter(client.Transport, cb)

	re := regexp.MustCompile(`\d{5}`)

	// Opens after failure ratio exceeded.
	for range cb.FailureThreshold + 1 {
		_, err := client.Get(ts.URL)
		msg := re.ReplaceAllString(err.Error(), "8080")
		fmt.Println(msg)
	}

	// Output:
	// initial status:
	// closed
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
	// Get "http://127.0.0.1:8080": circuit-breaker: broken
}
