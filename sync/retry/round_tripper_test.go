package retry_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestRoundTripperPost(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input string) (string, error) {
		var attempts int
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			body, _ := io.ReadAll(r.Body)
			evaltest.Log(ctx, evaltest.NewT[any]("request", nil, string(body), nil))
			if attempts < 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write(body)
		}))
		defer ts.Close()

		client := ts.Client()
		client.Transport = retry.NewRoundTripper(client.Transport, nil, retry.N(3), retry.NoWait)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL, strings.NewReader(input))
		if err != nil {
			return "", err
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		return string(b), err
	})
}
