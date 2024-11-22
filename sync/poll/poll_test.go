package poll_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/alextanhongpin/core/sync/poll"
)

func TestPoll(t *testing.T) {
	p := poll.New()

	ch, stop := p.Poll(func(ctx context.Context) error {
		return poll.EOQ
	})

	for msg := range ch {
		t.Logf("%+v\n", msg)
		if errors.Is(msg.Err, poll.EOQ) {
			stop()
		}
	}
}

func TestFailure(t *testing.T) {
	p := poll.New()
	p.FailureThreshold = 3

	ch, stop := p.Poll(func(ctx context.Context) error {
		return errors.New("bad request")
	})

	for msg := range ch {
		t.Logf("%+v\n", msg)
		if errors.Is(msg.Err, poll.ErrLimitExceeded) {
			stop()
		}
	}
}

func TestChannel(t *testing.T) {
	p := poll.New()
	p.BatchSize = 3
	p.MaxConcurrency = 3

	var count atomic.Int64
	ch, stop := p.Poll(func(ctx context.Context) error {
		if count.Add(1) >= 10 {
			return poll.EOQ
		}

		return nil
	})

	for msg := range ch {
		t.Logf("%+v\n", msg)
		if errors.Is(msg.Err, poll.EOQ) {
			stop()
		}
	}
}

func TestIdle(t *testing.T) {
	p := poll.New()
	p.BatchSize = 3
	p.MaxConcurrency = 3

	ch, stop := p.Poll(func(ctx context.Context) error {
		return poll.EOQ
	})

	var count atomic.Int64
	for msg := range ch {
		t.Logf("%+v\n", msg)

		if errors.Is(msg.Err, poll.EOQ) {
			if count.Add(1) > 2 {
				stop()
			}
		}
	}
}
