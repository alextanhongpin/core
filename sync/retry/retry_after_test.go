package retry_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func retryAfterResponse(code int, value string) *http.Response {
	h := http.Header{}
	if value != "" {
		h.Set("Retry-After", value)
	}
	return &http.Response{StatusCode: code, Body: http.NoBody, Header: h}
}

// elapsedBetweenCalls runs a GET that fails once with the given Retry-After
// value and returns the gap between the two transport calls.
func elapsedBetweenCalls(t *testing.T, value string, opts ...retry.RoundTripperOption) time.Duration {
	t.Helper()
	var times []time.Time
	tr := transportFunc(func(*http.Request) (*http.Response, error) {
		times = append(times, time.Now())
		if len(times) == 1 {
			return retryAfterResponse(503, value), nil
		}
		return retryAfterResponse(200, ""), nil
	})
	req, _ := http.NewRequest(http.MethodGet, "http://example.test", nil)
	got, err := retry.NewRoundTripper(tr, newRetry(t, 1), opts...).RoundTrip(req)
	if err != nil || got.StatusCode != 200 || len(times) != 2 {
		t.Fatalf("response=%v err=%v calls=%d", got, err, len(times))
	}
	return times[1].Sub(times[0])
}

func TestHTTPRetryAfterHonored(t *testing.T) {
	const limit = 80 * time.Millisecond
	future := time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)
	for _, value := range []string{"3600", "99999999999999999999", future} {
		if got := elapsedBetweenCalls(t, value, retry.WithRetryAfter(limit)); got < limit {
			t.Fatalf("Retry-After %q: waited %v, want at least %v", value, got, limit)
		}
	}
}

func TestHTTPRetryAfterIgnored(t *testing.T) {
	const limit = 80 * time.Millisecond
	past := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	for _, value := range []string{"", "soon", "-5", "0", past} {
		if got := elapsedBetweenCalls(t, value, retry.WithRetryAfter(limit)); got >= limit {
			t.Fatalf("Retry-After %q: waited %v", value, got)
		}
	}
	// Opt-in: without the option a huge Retry-After adds no wait.
	if got := elapsedBetweenCalls(t, "3600"); got >= limit {
		t.Fatalf("default waited %v", got)
	}
	if got := elapsedBetweenCalls(t, "3600", retry.WithRetryAfter(0)); got >= limit {
		t.Fatalf("zero max waited %v", got)
	}
}

func TestHTTPRetryAfterIsMaxOfBackoffAndHeader(t *testing.T) {
	r, err := retry.New(retry.Config{MaxRetries: 1, Backoff: retry.NewConstantBackoff(120 * time.Millisecond), Throttler: retry.NewNoopThrottler()})
	if err != nil {
		t.Fatal(err)
	}
	var times []time.Time
	tr := transportFunc(func(*http.Request) (*http.Response, error) {
		times = append(times, time.Now())
		return retryAfterResponse(503, "1"), nil
	})
	req, _ := http.NewRequest(http.MethodGet, "http://example.test", nil)
	// Retry-After (capped at 50ms) is shorter than the 120ms backoff: no extra wait.
	got, err := retry.NewRoundTripper(tr, r, retry.WithRetryAfter(50*time.Millisecond)).RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	got.Body.Close()
	gap := times[1].Sub(times[0])
	if gap < 120*time.Millisecond || gap >= 165*time.Millisecond {
		t.Fatalf("gap %v, want about the 120ms backoff", gap)
	}
}

func TestHTTPRetryAfterCancellationAndExhaustion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	calls := 0
	tr := transportFunc(func(*http.Request) (*http.Response, error) { calls++; return retryAfterResponse(503, "3600"), nil })
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test", nil)
	start := time.Now()
	got, err := retry.NewRoundTripper(tr, newRetry(t, 2), retry.WithRetryAfter(time.Minute)).RoundTrip(req)
	if got != nil || !errors.Is(err, retry.ErrCanceled) || calls != 1 || time.Since(start) > 2*time.Second {
		t.Fatalf("response=%v err=%v calls=%d", got, err, calls)
	}

	// The final response is returned immediately; no wait follows exhaustion.
	calls = 0
	req, _ = http.NewRequest(http.MethodGet, "http://example.test", nil)
	rt := retry.NewRoundTripper(tr, newRetry(t, 0), retry.WithRetryAfter(time.Minute))
	start = time.Now()
	got, err = rt.RoundTrip(req)
	if err != nil || got.StatusCode != 503 || calls != 1 || time.Since(start) > time.Second {
		t.Fatalf("response=%v err=%v calls=%d", got, err, calls)
	}
	got.Body.Close()
}

func TestHTTPDiscardedBodyIsDrainedUpTo4KiB(t *testing.T) {
	big := &trackedBody{Reader: strings.NewReader(strings.Repeat("x", 10000))}
	small := &trackedBody{Reader: strings.NewReader("tiny")}
	calls := 0
	tr := transportFunc(func(*http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			return &http.Response{StatusCode: 503, Body: big, Header: http.Header{}}, nil
		case 2:
			return &http.Response{StatusCode: 503, Body: small, Header: http.Header{}}, nil
		}
		return retryAfterResponse(200, ""), nil
	})
	req, _ := http.NewRequest(http.MethodGet, "http://example.test", nil)
	got, err := retry.NewRoundTripper(tr, newRetry(t, 2)).RoundTrip(req)
	if err != nil || got.StatusCode != 200 {
		t.Fatalf("response=%v err=%v", got, err)
	}
	rest, _ := io.ReadAll(big.Reader)
	if big.closed != 1 || len(rest) != 10000-4096 {
		t.Fatalf("big: closed=%d unread=%d", big.closed, len(rest))
	}
	rest, _ = io.ReadAll(small.Reader)
	if small.closed != 1 || len(rest) != 0 {
		t.Fatalf("small: closed=%d unread=%d", small.closed, len(rest))
	}
}
