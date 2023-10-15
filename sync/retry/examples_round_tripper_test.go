package retry_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRoundTripper() {
	ts := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, r *http.Request) {
		resp.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	opt := retry.NewOption()
	opt.Delay = 0

	r := retry.New[*http.Response](opt)
	r.ShouldHandle = func(resp *http.Response, err error) (bool, error) {
		// Skip if cancelled by caller.
		if errors.Is(err, context.Canceled) {
			return false, err
		}

		// Retry when status code is 5XX.
		if resp != nil && resp.StatusCode >= http.StatusInternalServerError {
			return true, errors.New(resp.Status)
		}

		return err != nil, err
	}
	r.OnRetry = func(e retry.Event) {
		fmt.Printf("retry.Event: %+v\n", e)
	}

	client := ts.Client()
	client.Transport = &retry.RoundTripper{
		Transport: client.Transport,
		Retrier:   r,
	}

	_, err := client.Get(ts.URL)
	if err != nil {
		// Replace port since it changes dynamically and breaks the test.
		re := regexp.MustCompile(`\d{5}`)
		msg := re.ReplaceAllString(err.Error(), "8080")
		fmt.Println(msg)
	}

	// Output:
	// retry.Event: {Attempt:1 Delay:0s Err:500 Internal Server Error}
	// retry.Event: {Attempt:2 Delay:0s Err:500 Internal Server Error}
	// retry.Event: {Attempt:3 Delay:0s Err:500 Internal Server Error}
	// retry.Event: {Attempt:4 Delay:0s Err:500 Internal Server Error}
	// retry.Event: {Attempt:5 Delay:0s Err:500 Internal Server Error}
	// retry.Event: {Attempt:6 Delay:0s Err:500 Internal Server Error}
	// retry.Event: {Attempt:7 Delay:0s Err:500 Internal Server Error}
	// retry.Event: {Attempt:8 Delay:0s Err:500 Internal Server Error}
	// retry.Event: {Attempt:9 Delay:0s Err:500 Internal Server Error}
	// retry.Event: {Attempt:10 Delay:0s Err:500 Internal Server Error}
	// Get "http://127.0.0.1:8080": retry: max attempts reached - retry 10 times, took 0s: 500 Internal Server Error
}
