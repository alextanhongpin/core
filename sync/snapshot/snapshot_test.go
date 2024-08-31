package snapshot_test

import (
	"context"
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
	bg, stop := snapshot.New(ctx, func(ctx context.Context, evt snapshot.Event) {
		events = append(events, evt)
	}, policies...)
	defer stop()

	bg.Inc(10_000)
	time.Sleep(10 * time.Millisecond)

	is := assert.New(t)
	is.Equal(snapshot.Event{Count: 10_000, Policy: policies[0]}, events[0])

	bg.Inc(1_000)
	time.Sleep(20 * time.Millisecond)
	is.Equal(snapshot.Event{Count: 1_000, Policy: policies[1]}, events[1])

	bg.Inc(100)
	time.Sleep(30 * time.Millisecond)
	is.Equal(snapshot.Event{Count: 100, Policy: policies[2]}, events[2])
}
