package circuitbreaker

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestDoAllowsConcurrentAndReentrantCalls(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cb := MustNew(Config{})
		started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
		go func() { defer close(done); cb.Do(func() error { close(started); <-release; return nil }) }()
		<-started
		if err := cb.Do(func() error {
			if cb.Status() != Closed {
				t.Error("unexpected status")
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		close(release)
		<-done
	})
}

func TestOldResultDoesNotChangeNewState(t *testing.T) {
	cb := MustNew(Config{})
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		cb.Do(func() error { close(started); <-release; return errors.New("old failure") })
	}()
	<-started
	cb.SetStatus(HalfOpen)
	close(release)
	<-done
	if cb.Status() != HalfOpen {
		t.Fatal("old failure changed half-open state")
	}
}

func TestSetOpenedStartsTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cb := MustNew(Config{})
		cb.SetStatus(Opened)
		if err := cb.Do(func() error { t.Fatal("opened callback ran"); return nil }); err != ErrOpened {
			t.Fatal(err)
		}
		time.Sleep(time.Minute)
		if err := cb.Do(func() error { return nil }); err != nil {
			t.Fatal(err)
		}
	})
}

type closeBody struct {
	io.Reader
	closed bool
}

func (b *closeBody) Close() error { b.closed = true; return nil }

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestTransportClosesDiscardedResponse(t *testing.T) {
	body := &closeBody{Reader: strings.NewReader("server error")}
	tr := NewTransporter(transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Status: "500 Internal Server Error", Body: body}, nil
	}), MustNew(Config{}))
	if resp, err := tr.RoundTrip(&http.Request{}); err == nil || resp != nil {
		t.Fatal("expected discarded response and error")
	}
	if !body.closed {
		t.Fatal("discarded response body left open")
	}
}

func TestTransportUsesDefaultWhenNil(t *testing.T) {
	tr := NewTransporter(nil, MustNew(Config{}))
	if tr.rt != http.DefaultTransport {
		t.Fatal("nil transport did not select default")
	}
}
