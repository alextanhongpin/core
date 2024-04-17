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
	http.StatusTooManyRequests,
	http.StatusInternalServerError,
	http.StatusBadGateway,
	http.StatusServiceUnavailable,
	http.StatusGatewayTimeout,
}

type transporter interface {
	RoundTrip(r *http.Request) (*http.Response, error)
}

type retrier interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) (*Result, error)
}

type RoundTripper struct {
	Transport transporter
	Retrier   retrier
}

func NewRoundTripper(t transporter, r retrier) *RoundTripper {
	return &RoundTripper{
		Transport: t,
		Retrier:   r,
	}
}

func (t *RoundTripper) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	_, retryErr := t.Retrier.Do(r.Context(), func(ctx context.Context) error {
		resp, err = t.Transport.RoundTrip(r)
		if err != nil {
			return err
		}

		if resp != nil && slices.Contains(retryableStatusCodes, resp.StatusCode) {
			return errors.New(resp.Status)
		}

		return nil
	})
	if allErr := errors.Join(retryErr, err); allErr != nil {
		return nil, allErr
	}

	return resp, nil
}
