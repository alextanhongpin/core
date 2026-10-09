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
	*broadcast.Broadcast[Policy]
	ch   chan int
	cfg  Config
	done chan struct{}
}

type Config struct {
	BufferSize int
	Policies   []Policy
}

func DefaultConfig() Config {
	return Config{
		BufferSize: 0,
		Policies:   DefaultPolicies(),
	}
}

// WithDefaults selects default policies only when Policies is nil. An explicit
// empty policy slice is invalid. Zero BufferSize keeps admission unbuffered.
func (cfg Config) WithDefaults() Config {
	if cfg.Policies == nil {
		cfg.Policies = DefaultPolicies()
	}
	return cfg
}

func (cfg Config) Validate() error {
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

// New owns a configuration copy including the policy slice. It validates before
// starting work. Stop interrupts pending notifications and waits for workers.
func New(cfg Config) (*Snapshot, func(), error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, nil, err
	}
	cfg.Policies = slices.Clone(cfg.Policies)
	slices.SortFunc(cfg.Policies, func(a, b Policy) int { return cmp.Compare(a.After, b.After) })
	b, stop := broadcast.New[Policy]()
	bg := &Snapshot{
		Broadcast: b,
		cfg:       cfg,
		ch:        make(chan int, cfg.BufferSize),
		done:      make(chan struct{}),
	}

	var wg sync.WaitGroup
	wg.Go(bg.loop)

	return bg, sync.OnceFunc(func() {
		close(bg.done)
		stop()
		wg.Wait()
	}), nil
}

// MustNew is New for startup wiring that must panic on invalid configuration.
func MustNew(cfg Config) (*Snapshot, func()) {
	s, stop, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return s, stop
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
	interval := minInterval(b.cfg.Policies)

	flush := func(n int) {
		count += n
		elapsed := time.Since(last)
		for _, p := range b.cfg.Policies {
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
