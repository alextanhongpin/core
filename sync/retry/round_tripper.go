package retry

import (
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

type transporter interface {
	RoundTrip(r *http.Request) (*http.Response, error)
}

type retrier interface {
	Do(func() error) error
}

type RoundTripper struct {
	Transport transporter
	retrier   retrier
}

func NewRoundTripper(t transporter, r retrier) *RoundTripper {
	return &RoundTripper{
		Transport: t,
		retrier:   r,
	}
}

func (t *RoundTripper) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	err = t.retrier.Do(func() error {
		resp, err = t.Transport.RoundTrip(r)
		if err != nil {
			return err
		}

		if resp != nil && slices.Contains(retryableStatusCodes, resp.StatusCode) {
			return errors.New(resp.Status)
		}

		return nil

	})
	if err != nil {
		return nil, err
	}

	return resp, err
}
