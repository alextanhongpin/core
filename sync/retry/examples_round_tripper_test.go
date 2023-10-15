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
	i := 0
	ts := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, r *http.Request) {
		i++
		fmt.Println("run", i)
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
	// run 1
	// run 2
	// run 3
	// run 4
	// run 5
	// run 6
	// run 7
	// run 8
	// run 9
	// run 10
	// run 11
	// Get "http://127.0.0.1:8080": retry: max attempts reached - retry 10 times, took 0s: 500 Internal Server Error
}
