package retry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

var retryableStatusCodes = []int{408, 425, 500, 502, 503, 504}

// maxDrain is how much of a discarded response body is read before closing it,
// so the connection can be reused without risking an unbounded read.
const maxDrain = 4 << 10

var _ http.RoundTripper = (*RoundTripper)(nil)

// RoundTripper retries eligible HTTP requests. Its configuration is immutable;
// supplied transports, runners, and callbacks must support concurrent calls.
type RoundTripper struct {
	rt                http.RoundTripper
	runner            Runner
	statusCodeHandler func(int) error
	retryableRequest  func(*http.Request) bool
	retryAfterMax     time.Duration // zero: Retry-After is ignored
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

// WithRetryAfter makes the adapter honor the Retry-After header (delta-seconds
// or HTTP-date) of a response that is about to be retried. The wait before the
// next attempt becomes max(backoff, Retry-After), where the requested wait is
// capped at maxWait. A nonpositive maxWait leaves the option off, which is the
// default. Missing, malformed, negative, or past values add no wait.
//
// The extra wait happens inside the attempt, after the runner's own backoff, so
// it counts against the request context and is interrupted by its cancellation.
// Only responses that the status handler marks retryable are considered, and
// 429 is not among the default retryable statuses: combine this option with
// WithStatusCodeHandler to retry 429 and 503 throttling responses. A final
// response returned on exhaustion or throttling never causes a wait.
func WithRetryAfter(maxWait time.Duration) RoundTripperOption {
	return func(rt *RoundTripper) {
		if maxWait > 0 {
			rt.retryAfterMax = maxWait
		}
	}
}

// parseRetryAfter returns the wait requested by a Retry-After value relative to
// now, capped at limit. ok is false when the value is absent or invalid.
func parseRetryAfter(v string, now time.Time, limit time.Duration) (d time.Duration, ok bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.ParseUint(v, 10, 63); err == nil {
		if secs > uint64(limit/time.Second) {
			return limit, true
		}
		return min(time.Duration(secs)*time.Second, limit), true
	} else if errors.Is(err, strconv.ErrRange) {
		return limit, true
	}
	t, err := http.ParseTime(v)
	if err != nil {
		return 0, false
	}
	return min(max(t.Sub(now), 0), limit), true
}

// discard drains a bounded prefix of the body, then closes it.
func discard(resp *http.Response) {
	io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrain))
	resp.Body.Close()
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
// attempts. Discarded responses have up to 4 KiB of body drained and are then
// closed, when the next attempt starts; a retryable response body therefore
// stays open until then. The caller closes the final body.
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
	var resumeAt time.Time // earliest start of the next attempt (WithRetryAfter)
	// A runner can reject without handing the original body to the transport.
	defer func() {
		if !invoked && req.Body != nil {
			req.Body.Close()
		}
	}()
	err := rt.runner.Do(req.Context(), func(ctx context.Context) error {
		if resp != nil {
			discard(resp)
			resp = nil
		}
		if !resumeAt.IsZero() {
			wait := time.Until(resumeAt)
			resumeAt = time.Time{}
			if wait > 0 {
				timer := time.NewTimer(wait)
				select {
				case <-ctx.Done():
					timer.Stop()
					return canceled(ctx)
				case <-timer.C:
				}
			}
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
		if err != nil && rt.retryAfterMax > 0 {
			if d, ok := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now(), rt.retryAfterMax); ok && d > 0 {
				resumeAt = time.Now().Add(d)
			}
		}
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
