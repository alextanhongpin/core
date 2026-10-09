package retry_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/sync/retry"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type trackedBody struct {
	io.Reader
	closed int
}

func (b *trackedBody) Close() error { b.closed++; return nil }
func response(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Body: body, Header: http.Header{"Retry-After": []string{"1"}}}
}

func TestHTTPRetryEligibility(t *testing.T) {
	for _, tc := range []struct {
		name, method  string
		replay, optIn bool
		want          int
	}{
		{"get", http.MethodGet, true, false, 2},
		{"post", http.MethodPost, true, false, 1},
		{"opt-in post", http.MethodPost, true, true, 2},
		{"stream", http.MethodPut, false, false, 1},
		{"opt-in stream", http.MethodPost, false, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, "http://example.test", strings.NewReader("payload"))
			original := &trackedBody{Reader: req.Body}
			req.Body = original
			if !tc.replay {
				req.GetBody = nil
			}
			calls := 0
			var bodies []*trackedBody
			tr := transportFunc(func(q *http.Request) (*http.Response, error) {
				b, err := io.ReadAll(q.Body)
				if err != nil {
					t.Fatal(err)
				}
				q.Body.Close()
				if string(b) != "payload" {
					t.Fatalf("replayed %q", b)
				}
				calls++
				body := &trackedBody{Reader: strings.NewReader("response")}
				bodies = append(bodies, body)
				if calls == 1 {
					return response(500, body), nil
				}
				return response(200, body), nil
			})
			var options []retry.RoundTripperOption
			if tc.optIn {
				options = append(options, retry.WithRetryableRequest(func(*http.Request) bool { return true }))
			}
			got, err := retry.NewRoundTripper(tr, newRetry(t, 1), options...).RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			if calls != tc.want || original.closed != 1 {
				t.Fatalf("calls=%d original closes=%d", calls, original.closed)
			}
			for i, b := range bodies {
				want := 1
				if i == len(bodies)-1 {
					want = 0
				}
				if b.closed != want {
					t.Fatalf("response %d closed %d times", i, b.closed)
				}
			}
			got.Body.Close()
		})
	}
}

func TestHTTPFinalResponse(t *testing.T) {
	for _, retries := range []int{0, 2} {
		calls := 0
		rt := retry.NewRoundTripper(transportFunc(func(q *http.Request) (*http.Response, error) {
			calls++
			return response(503, io.NopCloser(strings.NewReader("details"))), nil
		}), newRetry(t, retries))
		req, _ := http.NewRequest(http.MethodGet, "http://example.test", nil)
		got, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(got.Body)
		got.Body.Close()
		if calls != retries+1 || got.StatusCode != 503 || got.Header.Get("Retry-After") != "1" || string(b) != "details" {
			t.Fatalf("calls=%d response=%+v body=%s", calls, got, b)
		}
	}
}

func TestHTTPTransportFailureAndGetBodyFailure(t *testing.T) {
	for _, factoryFails := range []bool{false, true} {
		failure := errors.New("failure")
		original := &trackedBody{Reader: strings.NewReader("payload")}
		discarded := &trackedBody{Reader: strings.NewReader("response")}
		req, _ := http.NewRequest(http.MethodPut, "http://example.test", nil)
		req.Body = original
		req.GetBody = func() (io.ReadCloser, error) {
			if factoryFails {
				return nil, failure
			}
			return io.NopCloser(strings.NewReader("payload")), nil
		}
		calls := 0
		rt := retry.NewRoundTripper(transportFunc(func(q *http.Request) (*http.Response, error) {
			calls++
			q.Body.Close()
			if factoryFails {
				return response(503, discarded), nil
			}
			return nil, failure
		}), newRetry(t, 1))
		got, err := rt.RoundTrip(req)
		if got != nil || !errors.Is(err, failure) || original.closed != 1 {
			t.Fatalf("response=%v err=%v closes=%d", got, err, original.closed)
		}
		if factoryFails && (calls != 1 || discarded.closed != 1) {
			t.Fatalf("calls=%d response closes=%d", calls, discarded.closed)
		}
	}
}

func TestHTTPPreCanceledAndDerivedContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	original := &trackedBody{Reader: strings.NewReader("payload")}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPut, "http://example.test", nil)
	req.Body = original
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("payload")), nil }
	rt := retry.NewRoundTripper(transportFunc(func(*http.Request) (*http.Response, error) { t.Fatal("transport called"); return nil, nil }), newRetry(t, 1))
	_, err := rt.RoundTrip(req)
	if !errors.Is(err, context.Canceled) || original.closed != 1 {
		t.Fatalf("err=%v closes=%d", err, original.closed)
	}
	key := struct{}{}
	derived := context.WithValue(context.Background(), key, "derived")
	req, _ = http.NewRequest(http.MethodGet, "http://example.test", nil)
	rt = retry.NewRoundTripper(transportFunc(func(q *http.Request) (*http.Response, error) {
		if q.Context().Value(key) != "derived" {
			t.Fatal("derived context lost")
		}
		return response(200, http.NoBody), nil
	}), derivedRunner{derived})
	_, err = rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
}

type derivedRunner struct{ ctx context.Context }

func (r derivedRunner) Do(_ context.Context, fn func(context.Context) error) error { return fn(r.ctx) }

func TestHTTPThrottlingPreservesResponse(t *testing.T) {
	limiter, err := retry.NewThrottler(retry.ThrottlerConfig{MaxTokens: 1})
	if err != nil {
		t.Fatal(err)
	}
	// Exhaust the retry budget before sending the logical operation.
	limiter.Allow()
	r, err := retry.New(retry.Config{MaxRetries: 2, Throttler: limiter})
	if err != nil {
		t.Fatal(err)
	}
	body := &trackedBody{Reader: strings.NewReader("busy")}
	calls := 0
	rt := retry.NewRoundTripper(transportFunc(func(*http.Request) (*http.Response, error) { calls++; return response(503, body), nil }), r)
	req, _ := http.NewRequest(http.MethodGet, "http://example.test", nil)
	got, err := rt.RoundTrip(req)
	if err != nil || calls != 1 || got.StatusCode != 503 || body.closed != 0 {
		t.Fatalf("response=%v err=%v calls=%d closes=%d", got, err, calls, body.closed)
	}
	got.Body.Close()
}

func TestHTTPCancellationClosesFinalResponse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	body := &trackedBody{Reader: strings.NewReader("busy")}
	rt := retry.NewRoundTripper(transportFunc(func(*http.Request) (*http.Response, error) { cancel(); return response(503, body), nil }), newRetry(t, 2))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test", nil)
	got, err := rt.RoundTrip(req)
	if got != nil || !errors.Is(err, retry.ErrCanceled) || body.closed != 1 {
		t.Fatalf("response=%v err=%v closes=%d", got, err, body.closed)
	}
}

func TestHTTPStatusPolicy(t *testing.T) {
	calls := 0
	rt := retry.NewRoundTripper(transportFunc(func(*http.Request) (*http.Response, error) { calls++; return response(429, http.NoBody), nil }), newRetry(t, 1), retry.WithStatusCodeHandler(func(code int) error {
		if code == 429 {
			return errors.New("rate limited")
		}
		return nil
	}))
	req, _ := http.NewRequest(http.MethodGet, "http://example.test", nil)
	got, err := rt.RoundTrip(req)
	if err != nil || got.StatusCode != 429 || calls != 2 {
		t.Fatalf("response=%v err=%v calls=%d", got, err, calls)
	}
	got.Body.Close()
}
