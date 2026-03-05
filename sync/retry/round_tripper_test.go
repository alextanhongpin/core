package retry_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/sync/retry"
	"github.com/go-openapi/testify/assert"
)

func TestRoundTripper(t *testing.T) {
	t.Run("non-retryable", func(t *testing.T) {
		var count int
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count++
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer ts.Close()

		// Arrange.
		re := retry.New(retry.N(5), retry.NoWait)
		client := ts.Client()
		client.Transport = retry.NewRoundTripper(client.Transport, re)

		// Act.
		resp, err := ts.Client().Get(ts.URL)

		// Assert.
		is := assert.New(t)
		is.NoError(err)
		is.Equal(1, count)
		is.Equal(http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("retryable", func(t *testing.T) {
		var count int

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		// Arrange.
		re := retry.New(retry.N(5), retry.NoWait)
		client := ts.Client()
		client.Transport = retry.NewRoundTripper(client.Transport, re)

		// Act.
		resp, err := ts.Client().Get(ts.URL)

		// Assert.
		is := assert.New(t)
		is.ErrorContains(err, "retry: limit exceeded")
		is.Equal(6, count)
		is.Nil(resp)
	})
}
