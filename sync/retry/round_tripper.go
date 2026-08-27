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

func (rt *RoundTripper) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	err = rt.r.Do(req.Context(), func(ctx context.Context) error {
		// Clone the request for each attempt so a consumed body can be rewound via GetBody.
		attemptReq := req
		if req.GetBody != nil {
			body, getErr := req.GetBody()
			if getErr != nil {
				return getErr
			}
			attemptReq = req.Clone(ctx)
			attemptReq.Body = body
		}

		resp, err = rt.rt.RoundTrip(attemptReq)
		if err != nil {
			return err
		}

		if err = rt.StatusCodeHandler(resp.StatusCode); err != nil {
			// Close body to avoid connection leaks on retry.
			resp.Body.Close()
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func DefaultStatusCodeHandler(code int) error {
	if slices.Contains(retryableStatusCodes, code) {
		return fmt.Errorf("status code: %d", code)
	}
	return nil
}
