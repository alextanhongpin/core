package circuitbreaker

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func TestFailureWindowIsFixed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cb := MustNew(Config{FailureThreshold: 3, FailurePeriod: time.Second})
		fail := func() { cb.Do(func() error { return errors.New("x") }) }
		fail()
		time.Sleep(900 * time.Millisecond)
		fail()
		time.Sleep(900 * time.Millisecond) // t=1.8s: first window expired
		fail()
		if cb.Status() != Closed {
			t.Fatal("failures in different fixed windows must not accumulate")
		}
		fail()
		fail()
		if cb.Status() != Opened {
			t.Fatal("expected open after threshold within one window")
		}
	})
}

func TestHalfOpenMaxCalls(t *testing.T) {
	cb := MustNew(Config{HalfOpenMaxCalls: 2, SuccessThreshold: 10})
	cb.SetStatus(HalfOpen)
	started, release := make(chan struct{}, 2), make(chan struct{})
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Do(func() error { started <- struct{}{}; <-release; return nil })
		}()
	}
	<-started
	<-started
	if err := cb.Do(func() error { t.Error("ran beyond limit"); return nil }); err != ErrOpened {
		t.Fatalf("got %v, want ErrOpened", err)
	}
	close(release)
	wg.Wait()
	if err := cb.Do(func() error { return nil }); err != nil {
		t.Fatalf("slot not released: %v", err)
	}
}

func TestHalfOpenSlotReleasedOnPanic(t *testing.T) {
	cb := MustNew(Config{HalfOpenMaxCalls: 1, SuccessThreshold: 10})
	cb.SetStatus(HalfOpen)
	func() {
		defer func() { recover() }()
		cb.Do(func() error { panic("boom") })
	}()
	if err := cb.Do(func() error { return nil }); err != nil {
		t.Fatalf("slot leaked after panic: %v", err)
	}
}

func TestHalfOpenMaxCallsNegativeInvalid(t *testing.T) {
	if _, err := New(Config{HalfOpenMaxCalls: -1}); err == nil {
		t.Fatal("accepted negative HalfOpenMaxCalls")
	}
}

type drainBody struct {
	r      *strings.Reader
	closed bool
	read   int
}

func (b *drainBody) Read(p []byte) (int, error) { n, err := b.r.Read(p); b.read += n; return n, err }
func (b *drainBody) Close() error               { b.closed = true; return nil }

func TestTransportDrainsBoundedBeforeClose(t *testing.T) {
	body := &drainBody{r: strings.NewReader(strings.Repeat("x", 100<<10))}
	tr := NewTransporter(transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Status: "503", Body: body}, nil
	}), MustNew(Config{}))
	if _, err := tr.RoundTrip(&http.Request{}); err == nil {
		t.Fatal("expected error")
	}
	if !body.closed || body.read != 4<<10 {
		t.Fatalf("closed=%v read=%d, want closed and 4096 read", body.closed, body.read)
	}
}

func TestTransportIgnoresCallerCancellation(t *testing.T) {
	cb := MustNew(Config{FailureThreshold: 1})
	tr := NewTransporter(transportFunc(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	}), cb)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.invalid", nil)
	if _, err := tr.RoundTrip(req); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if cb.Status() != Closed {
		t.Fatal("caller cancellation counted as failure")
	}
	// Other errors still count.
	tr2 := NewTransporter(transportFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial")
	}), cb)
	tr2.RoundTrip(&http.Request{})
	if cb.Status() != Opened {
		t.Fatal("real failure not counted")
	}
}

func TestTransportCancellationIsNotAHalfOpenSuccess(t *testing.T) {
	cb := MustNew(Config{SuccessThreshold: 1, HalfOpenMaxCalls: 1})
	cb.SetStatus(HalfOpen)
	tr := NewTransporter(transportFunc(func(r *http.Request) (*http.Response, error) {
		return nil, r.Context().Err()
	}), cb)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.invalid", nil)
	if _, err := tr.RoundTrip(req); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if got := cb.Status(); got != HalfOpen {
		t.Fatalf("canceled probe changed status to %v", got)
	}
	// The probe slot must be released.
	if err := cb.Do(func() error { return nil }); err != nil {
		t.Fatalf("slot not released: %v", err)
	}
}
