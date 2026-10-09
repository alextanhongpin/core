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
	cfg := circuitbreaker.DefaultConfig()
	cfg.OpenTimeout = 100 * time.Millisecond
	cfg.FailureThreshold = 10
	cfg.FailurePeriod = time.Second
	cb := circuitbreaker.MustNew(cfg)

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
	for range cfg.FailureThreshold + 1 {
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
	// Get "http://127.0.0.1:8080": circuitbreaker: opened
}
