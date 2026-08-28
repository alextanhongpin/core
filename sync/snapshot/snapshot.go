// package snapshot implements redis-snapshot like mechanism - the higher the
// frequency, the more frequent the execution.
package snapshot

import (
	"cmp"
	"slices"
	"sync"
	"time"
)

type Policy struct {
	Changes int
	After   time.Duration
}

func NewOptions() []Policy {
	return []Policy{
		{Changes: 1_000, After: time.Second},
		{Changes: 100, After: 10 * time.Second},
		{Changes: 10, After: time.Minute},
		{Changes: 1, After: time.Hour},
	}
}

type Background struct {
	*Broadcast[Policy]
	policies []Policy
	ch       chan int
	done     chan struct{}
	interval time.Duration
}

type Config struct {
	Policies   []Policy
	Interval   time.Duration
	BufferSize int
}

func New(cfg Config) (*Background, func()) {
	interval := cmp.Or(cfg.Interval, minInterval(cfg.Policies))
	if interval == 0 {
		panic("snapshot: interval must be non-zero")
	}
	if len(cfg.Policies) == 0 {
		panic("snapshot: policies is empty")
	}
	slices.SortFunc(cfg.Policies, func(a, b Policy) int {
		return cmp.Compare(a.After, b.After)
	})
	b, stop := NewBroadcast[Policy]()
	bg := &Background{
		Broadcast: b,
		ch:        make(chan int, cfg.BufferSize),
		done:      make(chan struct{}),
		policies:  cfg.Policies,
		interval:  interval,
	}

	var wg sync.WaitGroup
	wg.Go(bg.loop)
	return bg, sync.OnceFunc(func() {
		close(bg.done)
		wg.Wait()
		stop()
	})
}

// Inc increments the counter by 1. Calls Add(1).
func (b *Background) Inc() {
	b.Add(1)
}

// Add increments the counter by n.
func (b *Background) Add(n int) {
	select {
	case <-b.done:
		return
	case b.ch <- n:
	}
}

func (b *Background) loop() {
	defer close(b.ch)

	var count int
	last := time.Now()
	interval := minInterval(b.policies)

	flush := func(n int) {
		count += n
		elapsed := time.Since(last)
		for _, p := range b.policies {
			if elapsed < p.After {
				return
			}
			if count >= p.Changes {
				count = 0
				last = time.Now()
				b.Send(p)
				return
			}
		}
	}
	defer flush(0)

	for {
		select {
		case <-b.done:
			return
		case <-time.After(interval):
			flush(0)
		case n := <-b.ch:
			flush(n)
		}
	}
}

func minInterval(policies []Policy) time.Duration {
	// Take the first non-zero duration.
	// It can be zero, essentially meaning always trigger when reach the amount.
	for _, p := range policies {
		if p.After != 0 {
			return p.After
		}
	}

	panic("snapshot: zero interval")
}
