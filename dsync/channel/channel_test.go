package channel_test

import (
	"context"
	"errors"
	"github.com/alextanhongpin/core/dsync/channel"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) { stop := redistest.Init(); code := m.Run(); stop(); os.Exit(code) }
func TestReplayAndRepeatedPayloads(t *testing.T) {
	client := redistest.Client(t)
	ch, err := channel.New(client)
	if err != nil {
		t.Fatal(err)
	}
	key := t.Name()
	for range 2 {
		if err := ch.Send(t.Context(), key, []byte("same")); err != nil {
			t.Fatal(err)
		}
	}
	first, err := ch.RecvAfter(t.Context(), key, "0-0", -1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ch.RecvAfter(t.Context(), key, first.ID, -1)
	if err != nil || first.ID == second.ID || string(second.Value) != "same" {
		t.Fatalf("%+v %v", second, err)
	}
}
func TestMalformedEntryReturnsError(t *testing.T) {
	client := redistest.Client(t)
	client.Do(t.Context(), "XADD", t.Name(), "*", "other", "value")
	ch := channel.MustNew(client)
	if _, err := ch.RecvAfter(t.Context(), t.Name(), "0-0", -1); err == nil {
		t.Fatal("invalid stream entry accepted")
	}
	if _, err := channel.New(nil); err == nil {
		t.Fatal("nil client accepted")
	}
}

func TestBlockingReadCancellation(t *testing.T) {
	client := redistest.Client(t)
	ch := channel.MustNew(client)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := ch.RecvAfter(ctx, t.Name(), "0-0", 0); result <- err }()
	// Wait until Redis has admitted the blocking read before canceling.
	deadline := time.Now().Add(time.Second)
	for {
		listing, err := client.ClientList(t.Context()).Result()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(listing, "cmd=xread") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("read did not reach Redis")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("read ignored cancellation")
	}
}
func TestPositiveReadDeadline(t *testing.T) {
	ch := channel.MustNew(redistest.Client(t))
	start := time.Now()
	if _, err := ch.RecvAfter(t.Context(), t.Name(), "0-0", 500*time.Microsecond); err == nil {
		t.Fatal("empty stream returned a value")
	}
	if time.Since(start) > time.Second {
		t.Fatal("short timeout became an infinite read")
	}
}
