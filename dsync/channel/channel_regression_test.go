package channel_test

import (
	"errors"
	"testing"

	"github.com/alextanhongpin/core/dsync/channel"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	redis "github.com/redis/go-redis/v9"
)

// A malformed entry used to be returned without its ID, so a cursor-based
// reader could never advance past it.
func TestMalformedEntryCanBeSkipped(t *testing.T) {
	client := redistest.Client(t)
	key := t.Name()
	if err := client.XAdd(t.Context(), &redis.XAddArgs{Stream: key, Values: []string{"other", "value"}}).Err(); err != nil {
		t.Fatal(err)
	}
	ch := channel.MustNew(client)
	if err := ch.Send(t.Context(), key, []byte("ok")); err != nil {
		t.Fatal(err)
	}
	bad, err := ch.RecvAfter(t.Context(), key, "0-0", -1)
	if !errors.Is(err, channel.ErrMalformedEntry) || bad.ID == "" {
		t.Fatalf("got %+v, %v; want ErrMalformedEntry with ID", bad, err)
	}
	next, err := ch.RecvAfter(t.Context(), key, bad.ID, -1)
	if err != nil || string(next.Value) != "ok" {
		t.Fatalf("got %+v, %v", next, err)
	}
}

func TestEmptyKeyRejected(t *testing.T) {
	ch := channel.MustNew(redistest.Client(t))
	if err := ch.Send(t.Context(), "", []byte("x")); !errors.Is(err, channel.ErrInvalidArgument) {
		t.Fatalf("Send: %v", err)
	}
	if _, err := ch.RecvAfter(t.Context(), "", "0-0", -1); !errors.Is(err, channel.ErrInvalidArgument) {
		t.Fatalf("RecvAfter: %v", err)
	}
	if _, err := ch.RecvAfter(t.Context(), "k", "", -1); !errors.Is(err, channel.ErrInvalidArgument) {
		t.Fatalf("RecvAfter: %v", err)
	}
}

func TestTrim(t *testing.T) {
	ch := channel.MustNew(redistest.Client(t))
	key := t.Name()
	for _, v := range []string{"a", "b", "c"} {
		if err := ch.Send(t.Context(), key, []byte(v)); err != nil {
			t.Fatal(err)
		}
	}
	n, err := ch.Trim(t.Context(), key, 1)
	if err != nil || n != 2 {
		t.Fatalf("Trim = %d, %v; want 2", n, err)
	}
	msg, err := ch.RecvAfter(t.Context(), key, "0-0", -1)
	if err != nil || string(msg.Value) != "c" {
		t.Fatalf("got %+v, %v; want c", msg, err)
	}
	if _, err := ch.Trim(t.Context(), key, -1); !errors.Is(err, channel.ErrInvalidArgument) {
		t.Fatalf("negative maxLen: %v", err)
	}
}
