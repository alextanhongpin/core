package retry

import (
	"cmp"
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
	rt                http.RoundTripper
	r                 retry
	StatusCodeHandler func(code int) error
}

func NewRoundTripper(rt http.RoundTripper, r retry) *RoundTripper {
	return &RoundTripper{
		rt:                cmp.Or(rt, http.DefaultTransport),
		r:                 r,
		StatusCodeHandler: DefaultStatusCodeHandler,
	}
}

func (rt *RoundTripper) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	err = rt.r.Do(r.Context(), func(ctx context.Context) error {
		resp, err = rt.RoundTrip(r)
		if err != nil {
			return err
		}

		if err = rt.StatusCodeHandler(resp.StatusCode); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return
}

func DefaultStatusCodeHandler(code int) error {
	if slices.Contains(retryableStatusCodes, code) {
		return fmt.Errorf("status code: %d", code)
	}
	return nil
}
