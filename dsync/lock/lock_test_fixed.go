// Example of how to fix race conditions in tests
package lock

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/lock"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"github.com/stretchr/testify/assert"
)

// Fixed version of TestLock_WaitSuccess without race conditions
func TestLock_WaitSuccess_Fixed(t *testing.T) {
	var (
		ch     = make(chan bool)
		client = redistest.Client(t)
		is     = assert.New(t)
		key    = t.Name()
		wg     sync.WaitGroup
	)

	// Use a mutex to protect the events slice
	var eventsMu sync.Mutex
	var events []string

	addEvent := func(event string) {
		eventsMu.Lock()
		events = append(events, event)
		eventsMu.Unlock()
	}

	getEvents := func() []string {
		eventsMu.Lock()
		defer eventsMu.Unlock()
		return append([]string{}, events...) // Return a copy
	}

	ctx := t.Context()

	wg.Add(2)
	go func() {
		defer wg.Done()

		err := lock.New(client).Do(ctx, key, func(ctx context.Context) error {
			addEvent("worker #1: lock acquired")
			close(ch)

			time.Sleep(100 * time.Millisecond)
			addEvent("worker #1: awake")
			return nil
		}, &lock.LockOption{
			Lock:         time.Second,
			Wait:         time.Second,
			RefreshRatio: 0.7,
		})
		is.NoError(err)
		addEvent("worker #1: done")
	}()

	go func() {
		defer wg.Done()

		<-ch

		err := lock.New(client).Do(ctx, key, func(ctx context.Context) error {
			addEvent("worker #2: lock acquired")
			return nil
		}, &lock.LockOption{
			Lock:         time.Second,
			Wait:         200 * time.Millisecond,
			RefreshRatio: 0.7,
		})
		addEvent("worker #2: done")
		is.NoError(err)
	}()

	wg.Wait()

	finalEvents := getEvents()
	is.Equal([]string{
		"worker #1: lock acquired",
		"worker #1: awake",
		"worker #1: done",
		"worker #2: lock acquired",
		"worker #2: done",
	}, finalEvents)
}
