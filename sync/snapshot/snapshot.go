// package snapshot implements redis-snapshot like mechanism - the higher the
// frequency, the more frequent the execution.
package snapshot

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/types/sliceutil"
	"golang.org/x/exp/slices"
)

type Policy struct {
	Every    int64
	Interval time.Duration
}

func (p Policy) IntervalSeconds() int64 {
	return p.Interval.Nanoseconds() / 1e9
}

type Manager struct {
	unix     atomic.Int64
	every    atomic.Int64
	policies []Policy
}

func NewPolicy(every int64, interval time.Duration) Policy {
	return Policy{
		Every:    every,
		Interval: interval,
	}
}

func New(policies []Policy) *Manager {
	if len(policies) == 0 {
		panic("snapshot: cannot instantiate new snapshot with no policies")
	}

	slices.SortFunc(policies, func(a, b Policy) bool {
		return a.IntervalSeconds() < b.IntervalSeconds()
	})

	return &Manager{
		policies: policies,
	}
}

func (m *Manager) Inc(n int64) int64 {
	return m.every.Add(n)
}

// Exec allows lazy execution.
func (m *Manager) Exec(ctx context.Context, h func(ctx context.Context)) {
	if !m.allow() {
		return
	}

	h(ctx)
	m.reset()
}

// Run executes whenever the condition is fulfilled. Returning an error will
// cause the every and timer not to reset.
// The client should be responsible for logging and handling the error.
func (m *Manager) Run(ctx context.Context, h func(ctx context.Context)) func() {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	wg.Add(1)

	stop := func() {
		cancel()

		wg.Wait()
	}

	go func() {
		defer cancel()
		defer wg.Done()

		m.start(ctx, h)
	}()

	return stop
}

func (m *Manager) start(ctx context.Context, h func(ctx context.Context)) {
	t := time.NewTicker(m.tick())
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.Exec(ctx, h)
		}
	}
}

func (m *Manager) reset() {
	m.every.Store(0)
	m.unix.Store(time.Now().Unix())
}

func (m *Manager) allow() bool {
	delta := time.Now().Unix() - m.unix.Load()
	every := m.every.Load()

	for _, p := range m.policies {
		if delta < p.IntervalSeconds() {
			return false
		}

		if every >= p.Every {
			return true
		}
	}

	return false
}

func (m *Manager) tick() time.Duration {
	intervals := sliceutil.Map(m.policies, func(i int) int64 {
		return m.policies[i].IntervalSeconds()
	})

	// Find the greatest common denominator to run the pooling.
	// If the given policy interval is 3s, 6s, and 9s for example,
	// the GCD will be 3s.
	gcd := internal.GCD(intervals)

	return time.Duration(gcd) * time.Second
}
