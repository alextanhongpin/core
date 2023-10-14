package retry_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleRoundTripper() {
	backoffs := retry.Backoffs{0, 0, 0, 0, 0}

	i := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i++
		fmt.Println("run", i)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := ts.Client()
	client.Transport = &retry.RoundTripper{
		Transport: client.Transport,
		Backoffs:  backoffs,
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
	// Get "http://127.0.0.1:8080": 500 Internal Server Error
}
