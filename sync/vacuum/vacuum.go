package vacuum

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

type Handler func(ctx context.Context)

func (h Handler) Exec(ctx context.Context) {
	h(ctx)
}

type Vacuum struct {
	unix     atomic.Int64
	every    atomic.Int64
	policies []Policy
	tick     time.Duration
	begin    sync.Once
}

func NewPolicy(every int64, interval time.Duration) Policy {
	return Policy{
		Every:    every,
		Interval: interval,
	}
}

func New(policies []Policy) *Vacuum {
	if len(policies) == 0 {
		panic("vacuum: cannot instantiate new vacuum with no policies")
	}

	slices.SortFunc(policies, func(a, b Policy) bool {
		return a.IntervalSeconds() < b.IntervalSeconds()
	})

	intervals := sliceutil.Map(policies, func(i int) int64 {
		return policies[i].IntervalSeconds()
	})

	// Find the greatest common denominator to run the pooling.
	// If the given policy interval is 3s, 6s, and 9s for example,
	// the GCD will be 3s.
	gcd := internal.GCD(intervals)
	tick := time.Duration(gcd) * time.Second

	return &Vacuum{
		tick:     tick,
		policies: policies,
	}
}

func (v *Vacuum) Inc(n int64) int64 {
	return v.every.Add(n)
}

// Run executes whenever the condition is fulfilled. Returning an error will
// cause the every and timer not to reset.
// The client should be responsible for logging and handling the error.
func (v *Vacuum) Run(ctx context.Context, h Handler) (stop func()) {
	v.begin.Do(func() {
		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(ctx)
		wg.Add(1)

		stop = func() {
			cancel()

			wg.Wait()
		}

		go func() {
			defer cancel()
			defer wg.Done()

			v.start(ctx, h)
		}()
	})

	return
}

func (v *Vacuum) start(ctx context.Context, h Handler) {
	t := time.NewTicker(v.tick)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if !v.allow() {
				continue
			}

			h.Exec(ctx)
			v.reset()
		}
	}
}

func (v *Vacuum) reset() {
	v.every.Store(0)
	v.unix.Store(time.Now().Unix())
}

func (v *Vacuum) allow() bool {
	delta := time.Now().Unix() - v.unix.Load()
	every := v.every.Load()

	for _, p := range v.policies {
		if delta < p.IntervalSeconds() {
			return false
		}

		if every >= p.Every {
			return true
		}
	}

	return false
}
