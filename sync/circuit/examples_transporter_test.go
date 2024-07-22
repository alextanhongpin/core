package circuit_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"time"

	"github.com/alextanhongpin/core/sync/circuit"
)

func ExampleRoundTripper() {
	opt := circuit.NewOption()
	opt.BreakDuration = 100 * time.Millisecond
	opt.SamplingDuration = 1 * time.Second
	cb := circuit.New(opt)

	fmt.Println("initial status:")
	fmt.Println(cb.Status())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := ts.Client()
	client.Transport = circuit.NewTransporter(client.Transport, cb)

	// Opens after failure ratio exceeded.
	for range opt.FailureThreshold {
		resp, err := client.Get(ts.URL)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		fmt.Println(resp.Status)
	}
	_, err := client.Get(ts.URL)

	re := regexp.MustCompile(`\d{5}`)
	msg := re.ReplaceAllString(err.Error(), "8080")
	fmt.Println(msg)

	// Output:
	// initial status:
	// closed
	// 500 Internal Server Error
	// 500 Internal Server Error
	// 500 Internal Server Error
	// 500 Internal Server Error
	// 500 Internal Server Error
	// 500 Internal Server Error
	// 500 Internal Server Error
	// 500 Internal Server Error
	// 500 Internal Server Error
	// 500 Internal Server Error
	// Get "http://127.0.0.1:8080": circuit-breaker: broken
}
