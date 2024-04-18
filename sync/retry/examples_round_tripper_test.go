package retry_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRoundTripper() {
	status := http.StatusUnauthorized
	ts := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, r *http.Request) {
		resp.WriteHeader(status)
	}))
	defer ts.Close()

	r := retry.New(
		10*time.Millisecond,
		10*time.Millisecond,
		10*time.Millisecond,
		10*time.Millisecond,
		10*time.Millisecond,
	)
	r.Now = func() time.Time {
		return time.Time{}
	}
	r.OnRetry = func(e retry.Event) {
		fmt.Printf("retry.Event: %+v\n", e)
	}

	client := ts.Client()
	client.Transport = retry.NewRoundTripper(client.Transport, r)

	// No retry when 401.
	resp, err := client.Get(ts.URL)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp.Status)

	// Retry when 5xx.
	status = http.StatusInternalServerError
	_, err = client.Get(ts.URL)
	if err != nil {
		// Replace port since it changes dynamically and breaks the test.
		re := regexp.MustCompile(`\d{5}`)
		msg := re.ReplaceAllString(err.Error(), "8080")
		fmt.Println(msg)
	}

	// Output:
	// 401 Unauthorized
	// retry.Event: {StartAt:0001-01-01 00:00:00 +0000 UTC RetryAt:0001-01-01 00:00:00 +0000 UTC Attempt:1 Delay:7.281668ms Err:500 Internal Server Error}
	// retry.Event: {StartAt:0001-01-01 00:00:00 +0000 UTC RetryAt:0001-01-01 00:00:00 +0000 UTC Attempt:2 Delay:8.43475ms Err:500 Internal Server Error}
	// retry.Event: {StartAt:0001-01-01 00:00:00 +0000 UTC RetryAt:0001-01-01 00:00:00 +0000 UTC Attempt:3 Delay:9.099423ms Err:500 Internal Server Error}
	// retry.Event: {StartAt:0001-01-01 00:00:00 +0000 UTC RetryAt:0001-01-01 00:00:00 +0000 UTC Attempt:4 Delay:7.901345ms Err:500 Internal Server Error}
	// retry.Event: {StartAt:0001-01-01 00:00:00 +0000 UTC RetryAt:0001-01-01 00:00:00 +0000 UTC Attempt:5 Delay:5.640357ms Err:500 Internal Server Error}
	// Get "http://127.0.0.1:8080": retry: too many attempts
	// 500 Internal Server Error
}
