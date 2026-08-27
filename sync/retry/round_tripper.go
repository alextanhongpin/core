package retry

import (
	"context"
	"fmt"
	"net/http"
	"slices"
)

var retryableStatusCodes = []int{
	http.StatusRequestTimeout,
	http.StatusTooEarly,
	http.StatusInternalServerError,
	http.StatusBadGateway,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
}

var _ http.RoundTripper = (*RoundTripper)(nil)

type RoundTripper struct {
	fn func(ctx context.Context, r *http.Request) (*http.Response, error)
}

func StatusCodeHandler(code int) error {
	if slices.Contains(retryableStatusCodes, code) {
		return fmt.Errorf("status code: %d", code)
	}
	return nil
}

func NewRoundTripper(rt http.RoundTripper, statusCodeHandler func(statusCode int) error, opts ...Option) *RoundTripper {
	if statusCodeHandler == nil {
		statusCodeHandler = StatusCodeHandler
	}
	return &RoundTripper{
		fn: Handler(func(ctx context.Context, r *http.Request) (*http.Response, error) {
			resp, err := rt.RoundTrip(r)
			if err != nil {
				// This is transport error, don't retry.
				return nil, err
			}

			err = statusCodeHandler(resp.StatusCode)
			if err != nil {
				return nil, err
			}

			return resp, nil
		}, opts...),
	}
}

func (rt *RoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt.fn(r.Context(), r)
}
