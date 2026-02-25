package retry

import (
	"errors"
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

type RoundTripper struct {
	Transport  http.RoundTripper
	MaxRetries int
	StatusCode func(code int) error
	retry      retry
}

func NewRoundTripper(rt http.RoundTripper, r retry) *RoundTripper {
	return &RoundTripper{
		Transport:  rt,
		MaxRetries: 10,
		StatusCode: func(code int) error {
			if slices.Contains(retryableStatusCodes, code) {
				return errors.New(fmt.Sprint(code))
			}

			return nil
		},
		retry: r,
	}
}

func (t *RoundTripper) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	for _, err := range t.retry.Try(r.Context(), t.MaxRetries) {
		if err != nil {
			return nil, err
		}

		resp, err = t.Transport.RoundTrip(r)
		if err != nil {
			return nil, err
		}

		if err := t.StatusCode(resp.StatusCode); err != nil {
			return nil, err
		}

		return resp, nil
	}

	return nil, errors.ErrUnsupported
}
