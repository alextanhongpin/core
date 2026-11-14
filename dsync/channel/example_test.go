package channel_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/dsync/channel"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	redis "github.com/redis/go-redis/v9"
)

func newExampleClient() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: redistest.Addr()})
}

// Replay every entry from the start of a stream, remembering the cursor so
// that reading can resume later.
func ExampleChannel_RecvAfter() {
	client := newExampleClient()
	defer client.Close()
	ctx := context.Background()
	ch := channel.MustNew(client)
	key := "example:replay"
	defer ch.Close(ctx, key)

	for _, s := range []string{"created", "paid", "shipped"} {
		if err := ch.Send(ctx, key, []byte(s)); err != nil {
			panic(err)
		}
	}

	cursor := "0-0" // oldest entry; persist msg.ID to resume after a restart
	for {
		msg, err := ch.RecvAfter(ctx, key, cursor, -1) // -1: do not wait
		if errors.Is(err, redis.Nil) {
			break // caught up
		}
		if err != nil {
			panic(err)
		}
		fmt.Println(string(msg.Value))
		cursor = msg.ID
	}
	// Output:
	// created
	// paid
	// shipped
}

// Two independent readers both receive the same entry: reading does not
// consume it.
func ExampleChannel_Send_broadcast() {
	client := newExampleClient()
	defer client.Close()
	ctx := context.Background()
	ch := channel.MustNew(client)
	key := "example:broadcast"
	defer ch.Close(ctx, key)

	if err := ch.Send(ctx, key, []byte("cache invalidated")); err != nil {
		panic(err)
	}
	for _, reader := range []string{"reader-1", "reader-2"} {
		msg, err := ch.RecvAfter(ctx, key, "0-0", time.Second)
		if err != nil {
			panic(err)
		}
		fmt.Println(reader, "got", string(msg.Value))
	}
	// Output:
	// reader-1 got cache invalidated
	// reader-2 got cache invalidated
}

// Wait for the next new entry; entries that already exist are skipped.
func ExampleChannel_Recv() {
	client := newExampleClient()
	defer client.Close()
	ctx := context.Background()
	ch := channel.MustNew(client)
	key := "example:recv"
	defer ch.Close(ctx, key)

	_ = ch.Send(ctx, key, []byte("old")) // already present, not returned

	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = ch.Send(ctx, key, []byte("new"))
	}()
	value, err := ch.Recv(ctx, key, 5*time.Second)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(value))

	// With nothing new to read, a bounded wait ends with redis.Nil.
	_, err = ch.Recv(ctx, key, 10*time.Millisecond)
	fmt.Println(errors.Is(err, redis.Nil))
	// Output:
	// new
	// true
}

// Bound the stream's memory by keeping only the newest entries.
func ExampleChannel_Trim() {
	client := newExampleClient()
	defer client.Close()
	ctx := context.Background()
	ch := channel.MustNew(client)
	key := "example:trim"
	defer ch.Close(ctx, key)

	for i := range 5 {
		_ = ch.Send(ctx, key, fmt.Appendf(nil, "event %d", i))
	}
	removed, err := ch.Trim(ctx, key, 2)
	if err != nil {
		panic(err)
	}
	fmt.Println("removed", removed)
	msg, _ := ch.RecvAfter(ctx, key, "0-0", -1)
	fmt.Println("oldest kept:", string(msg.Value))
	// Output:
	// removed 3
	// oldest kept: event 3
}
