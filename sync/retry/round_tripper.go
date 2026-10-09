package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
)

var retryableStatusCodes = []int{408, 425, 500, 502, 503, 504}

var _ http.RoundTripper = (*RoundTripper)(nil)

// RoundTripper retries eligible HTTP requests. Its configuration is immutable;
// supplied transports, runners, and callbacks must support concurrent calls.
type RoundTripper struct {
	rt                http.RoundTripper
	runner            Runner
	statusCodeHandler func(int) error
	retryableRequest  func(*http.Request) bool
}

// RoundTripperOption configures an HTTP adapter at construction.
type RoundTripperOption func(*RoundTripper)

// WithStatusCodeHandler selects responses eligible for retry. A non-nil error
// asks the runner to retry. The final response is returned with a nil error.
// A nil handler selects DefaultStatusCodeHandler.
func WithStatusCodeHandler(fn func(int) error) RoundTripperOption {
	return func(rt *RoundTripper) {
		if fn != nil {
			rt.statusCodeHandler = fn
		}
	}
}

// WithRetryableRequest replaces the idempotent-method policy. Use it to opt
// into retrying mutations only when the server guarantees idempotency.
// Body replayability is required independently of this policy.
func WithRetryableRequest(fn func(*http.Request) bool) RoundTripperOption {
	return func(rt *RoundTripper) {
		if fn != nil {
			rt.retryableRequest = fn
		}
	}
}

// NewRoundTripper defaults nil transport to http.DefaultTransport and nil runner
// to a new Retry using DefaultConfig. Configure options before sharing the adapter.
func NewRoundTripper(transport http.RoundTripper, runner Runner, options ...RoundTripperOption) *RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	if runner == nil {
		runner, _ = New(DefaultConfig())
	}
	rt := &RoundTripper{rt: transport, runner: runner, statusCodeHandler: DefaultStatusCodeHandler, retryableRequest: defaultRetryableRequest}
	for _, option := range options {
		option(rt)
	}
	return rt
}

func defaultRetryableRequest(req *http.Request) bool {
	switch req.Method {
	case "", http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

// RoundTrip uses the original body on the first attempt and GetBody for later
// attempts. Discarded responses are closed; the caller closes the final body.
// Status exhaustion returns the final response with nil error, preserving HTTP
// status, headers, and body. Cancellation and transport failures return errors.
// Runner must obey its synchronous, sequential callback contract.
func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := canceled(req.Context()); err != nil {
		if req.Body != nil {
			req.Body.Close()
		}
		return nil, err
	}
	if !rt.retryableRequest(req) || (req.Body != nil && req.Body != http.NoBody && req.GetBody == nil) {
		return rt.rt.RoundTrip(req)
	}
	var resp *http.Response
	var invoked, lastStatus bool
	// A runner can reject without handing the original body to the transport.
	defer func() {
		if !invoked && req.Body != nil {
			req.Body.Close()
		}
	}()
	err := rt.runner.Do(req.Context(), func(ctx context.Context) error {
		if resp != nil {
			resp.Body.Close()
			resp = nil
		}
		lastStatus = false
		attemptReq := req.Clone(ctx)
		if invoked && req.Body != nil && req.Body != http.NoBody {
			body, err := req.GetBody()
			if err != nil {
				return err
			}
			attemptReq.Body = body
		}
		invoked = true
		var err error
		resp, err = rt.rt.RoundTrip(attemptReq)
		if err != nil {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			resp = nil
			return err
		}
		err = rt.statusCodeHandler(resp.StatusCode)
		lastStatus = err != nil
		return err
	})
	if err != nil {
		if resp != nil && lastStatus && req.Context().Err() == nil && !errors.Is(err, ErrCanceled) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			return resp, nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		return nil, err
	}
	return resp, nil
}

// DefaultStatusCodeHandler retries 408, 425, 500, 502, 503, and 504.
func DefaultStatusCodeHandler(code int) error {
	if slices.Contains(retryableStatusCodes, code) {
		return fmt.Errorf("status code: %d", code)
	}
	return nil
}
