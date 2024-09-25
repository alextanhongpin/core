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

	r := retry.New(retry.NewConstantBackOff(time.Millisecond))

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
	// Get "http://127.0.0.1:8080": 500
}
