package retry_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRoundTripper() {
	status := http.StatusUnauthorized
	ts := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, r *http.Request) {
		resp.WriteHeader(status)
	}))
	defer ts.Close()

	opt := retry.NewOption()
	opt.Delay = 0

	r := retry.New(opt)
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
	// Get "http://127.0.0.1:8080": retry: max attempts reached
	// 500 Internal Server Error
}
