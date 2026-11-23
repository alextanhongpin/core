//go:build unix

package server_test

import (
	"io"
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/alextanhongpin/core/http/server"
)

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func TestWaitGroupReturnsOnListenError(t *testing.T) {
	// Regression: a server that failed to listen was only logged, and
	// WaitGroup blocked until a signal arrived.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	ok := server.New(freeAddr(t), http.NotFoundHandler())
	busy := server.New(l.Addr().String(), http.NotFoundHandler())

	done := make(chan error, 1)
	go func() { done <- server.WaitGroup(ok, busy) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("WaitGroup = nil, want listen error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitGroup did not return after a listen failure")
	}
}

func TestWaitGroupDrainsInFlightRequests(t *testing.T) {
	// Regression: request contexts were the signal context, so every
	// in-flight request was canceled as soon as SIGTERM arrived.
	started := make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-time.After(300 * time.Millisecond):
			io.WriteString(w, "done")
		case <-r.Context().Done():
			http.Error(w, "canceled", http.StatusServiceUnavailable)
		}
	})

	addr := freeAddr(t)
	done := make(chan error, 1)
	go func() { done <- server.WaitGroup(server.New(addr, h)) }()

	var resp *http.Response
	var err error
	respc := make(chan struct{})
	go func() {
		defer close(respc)
		for range 50 {
			resp, err = http.Get("http://" + addr)
			if err == nil {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()

	<-started
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	<-respc
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "done" {
		t.Fatalf("got %d %q, want 200 done", resp.StatusCode, body)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("WaitGroup = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitGroup did not return after SIGTERM")
	}
}
