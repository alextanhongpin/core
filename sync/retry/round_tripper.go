package retry

import (
	"context"
	"errors"
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

type RoundTripper struct {
	http.RoundTripper
	StatusCodeHandler func(code int) error
	Options           []Option
}

func NewRoundTripper(rt http.RoundTripper, opts ...Option) *RoundTripper {
	return &RoundTripper{
		RoundTripper: rt,
		StatusCodeHandler: func(code int) error {
			// NOTE: We need to convert the status code into errors in order to retry it.
			if slices.Contains(retryableStatusCodes, code) {
				return errors.New(http.StatusText(code))
			}

			return nil
		},
		Options: opts,
	}
}

func (t *RoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return Do(r.Context(), func(context.Context) (*http.Response, error) {
		resp, err := t.RoundTripper.RoundTrip(r)
		if err != nil {
			// This is transport error, don't retry.
			return nil, err
		}

		err = t.StatusCodeHandler(resp.StatusCode)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}, t.Options...)
}
