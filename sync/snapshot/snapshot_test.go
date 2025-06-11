package snapshot_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/snapshot"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestSnapshot(t *testing.T) {
	policies := []snapshot.Policy{
		{Every: 10_000, Interval: 10 * time.Millisecond},
		{Every: 1_000, Interval: 20 * time.Millisecond},
		{Every: 100, Interval: 30 * time.Millisecond},
	}
	var events []snapshot.Event
	var mu sync.Mutex
	bg, stop := snapshot.New(ctx, func(ctx context.Context, evt snapshot.Event) {
		mu.Lock()
		events = append(events, evt)
		mu.Unlock()
	}, policies...)
	defer stop()

	bg.Inc(10_000)
	time.Sleep(11 * time.Millisecond)
	bg.Inc(1_000)
	time.Sleep(21 * time.Millisecond)
	bg.Inc(100)
	time.Sleep(31 * time.Millisecond)

	mu.Lock()
	is := assert.New(t)
	is.Equal(snapshot.Event{Count: 10_000, Policy: policies[0]}, events[0])
	is.Equal(snapshot.Event{Count: 1_000, Policy: policies[1]}, events[1])
	is.Equal(snapshot.Event{Count: 100, Policy: policies[2]}, events[2])
	mu.Unlock()
}
