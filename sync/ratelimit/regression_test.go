package ratelimit

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"
)

func TestFixedWindowClearAndOverflow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := MustNewFixedWindow(Config{Limit: math.MaxInt, Period: time.Second})
		if !r.AllowN("key", math.MaxInt) {
			t.Fatal("initial allowance denied")
		}
		if r.AllowN("key", math.MaxInt) {
			t.Fatal("overflow bypassed limit")
		}
		time.Sleep(time.Second)
		r.Clear()
		if r.Size() != 0 {
			t.Fatal("expired entry retained")
		}
	})
}
func TestGCRABatchAdmission(t *testing.T) {
	r := MustNewGCRA(Config{Limit: 1, Period: time.Hour, Burst: 2})
	if r.AllowN("oversize", 4) {
		t.Fatal("oversized batch admitted")
	}
	if !r.AllowN("key", 2) {
		t.Fatal("batch within burst denied")
	}
	if r.AllowN("key", 2) {
		t.Fatal("partial batch admitted")
	}
	if !r.Allow("key") {
		t.Fatal("rejected batch consumed tokens")
	}
}
func TestGCRARejectsSubNanosecondInterval(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected invalid interval panic")
		}
	}()
	MustNewGCRA(Config{Limit: 2, Period: time.Nanosecond})
}
func TestHTTPRetryAfterSeconds(t *testing.T) {
	r := MustNewFixedWindow(Config{Limit: 1, Period: 1500 * time.Millisecond})
	h := HTTP(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), FuncConfig[*http.Request]{RateLimiter: r, KeyFn: func(_ context.Context, _ *http.Request) (string, error) { return "key", nil }})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "2" {
		t.Fatalf("invalid retry response: %d %q", rec.Code, rec.Header().Get("Retry-After"))
	}
}
