package retry

import (
	"context"
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

type retryHandler interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type RoundTripper struct {
	Transport  http.RoundTripper
	MaxRetries int
	StatusCode func(code int) error
	Retry      retryHandler
}

func NewRoundTripper(rt http.RoundTripper, r retryHandler) *RoundTripper {
	return &RoundTripper{
		Transport:  rt,
		MaxRetries: 10,
		StatusCode: func(code int) error {
			// NOTE: We need to convert the status code into errors in order to retry it.
			if slices.Contains(retryableStatusCodes, code) {
				return errors.New(fmt.Sprint(code))
			}

			return nil
		},
		Retry: r,
	}
}

func (t *RoundTripper) RoundTrip(r *http.Request) (resp *http.Response, err error) {
	defer func() {
		if err != nil {
			resp = nil
		}
	}()
	err = t.Retry.Do(r.Context(), func(context.Context) error {
		resp, err = t.Transport.RoundTrip(r)
		if err != nil {
			return err
		}

		return t.StatusCode(resp.StatusCode)
	})
	return
}
