package retry_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alextanhongpin/core/sync/retry"
	"github.com/alextanhongpin/evaltest"
)

func TestRoundTripper(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, statusCode int) (any, error) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			evaltest.Log(ctx, evaltest.NewT[any]("request", nil, statusCode, nil))
			w.WriteHeader(statusCode)
		}))
		defer ts.Close()

		// Arrange.
		client := ts.Client()
		client.Transport = retry.NewRoundTripper(client.Transport, nil, retry.N(5), retry.NoWait)

		// Act.
		resp, err := ts.Client().Get(ts.URL)
		if resp != nil {
			return resp.StatusCode, err
		}
		return 0, err
	})
}
