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

type Snapshot struct {
	*Config
	*broadcast.Broadcast[Policy]
	ch       chan int
	policies []Policy
	done     chan struct{}
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
	if cfg == nil {
		return errors.New("snapshot: nil config")
	}
	if cfg.BufferSize < 0 {
		return errors.New("snapshot: negative buffer size")
	}
	for _, p := range cfg.Policies {
		if p.After < 0 || p.Changes <= 0 {
			return errors.New("snapshot: policies require nonnegative durations and positive changes")
		}
	}
	if len(cfg.Policies) == 0 {
		return errors.New("snapshot: no policies")
	}
	return nil
}

func New(cfg *Config) (*Snapshot, func()) {
	if err := cfg.Validate(); err != nil {
		panic(err)
	}
	owned := *cfg
	owned.Policies = slices.Clone(cfg.Policies)
	cfg = &owned
	slices.SortFunc(cfg.Policies, func(a, b Policy) int {
		return cmp.Compare(a.After, b.After)
	})
	b, stop := broadcast.New[Policy]()
	bg := &Snapshot{
		Broadcast: b,
		Config:    cfg,
		policies:  slices.Clone(cfg.Policies),
		ch:        make(chan int, cfg.BufferSize),
		done:      make(chan struct{}),
	}

	var wg sync.WaitGroup
	wg.Go(bg.loop)

	return bg, sync.OnceFunc(func() {
		close(bg.done)
		stop()
		wg.Wait()
	})
}

// Inc increments the counter by 1. Calls Add(1).
func (b *Snapshot) Inc() {
	b.Add(1)
}

// Add increments the counter by n.
func (b *Snapshot) Add(n int) {
	select {
	case <-b.done:
		return
	case b.ch <- n:
	}
}

func (b *Snapshot) loop() {

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

	var ticks <-chan time.Time
	if interval > 0 {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		ticks = ticker.C
	}
	for {
		select {
		case <-b.done:
			return
		case <-ticks:
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

	return 0
}
