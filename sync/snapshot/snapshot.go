// package snapshot implements redis-snapshot like mechanism - the higher the
// frequency, the more frequent the execution.
package snapshot

import (
	"cmp"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/alextanhongpin/core/sync/broadcast"
)

type Policy struct {
	After   time.Duration
	Changes int
}

func DefaultPolicies() []Policy {
	return []Policy{
		{Changes: 1_000, After: time.Second},
		{Changes: 100, After: 10 * time.Second},
		{Changes: 10, After: time.Minute},
		{Changes: 1, After: time.Hour},
	}
}

type Background struct {
	*Config
	*broadcast.Broadcast[Policy]
	ch   chan int
	done chan struct{}
}

type Config struct {
	BufferSize int
	Policies   []Policy
}

func DefaultConfig() *Config {
	return &Config{
		BufferSize: 0,
		Policies:   DefaultPolicies(),
	}
}

func (cfg *Config) Validate() error {
	if len(cfg.Policies) == 0 {
		return errors.New("snapshot: no policies")
	}
	return nil
}

func New(cfg *Config) (*Background, func()) {
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	slices.SortFunc(cfg.Policies, func(a, b Policy) int {
		return cmp.Compare(a.After, b.After)
	})
	b, stop := broadcast.New[Policy]()
	bg := &Background{
		Broadcast: b,
		Config:    cfg,
		ch:        make(chan int, cfg.BufferSize),
		done:      make(chan struct{}),
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
	interval := minInterval(b.Policies)

	flush := func(n int) {
		count += n
		elapsed := time.Since(last)
		for _, p := range b.Policies {
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
