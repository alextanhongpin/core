package vacuum

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/types/sliceutil"
	"golang.org/x/exp/slices"
)

var ErrClosed = errors.New("vacuum: closed")

type Policy struct {
	Every    int64
	Interval time.Duration
}

func (p Policy) IntervalSeconds() int64 {
	return p.Interval.Nanoseconds() / 1e9
}

type policy struct {
	every     int64
	threshold int64
}

type Handler func(ctx context.Context) error

func (h Handler) Exec(ctx context.Context) error {
	return h(ctx)
}

type Vacuum struct {
	unix     atomic.Int64
	every    atomic.Int64
	policies []policy
	tick     time.Duration
	done     chan struct{}
	begin    sync.Once
	end      sync.Once
	wg       sync.WaitGroup
}

func NewPolicy(every int64, interval time.Duration) Policy {
	return Policy{
		Every:    every,
		Interval: interval,
	}
}

func New(opts []Policy) *Vacuum {
	if len(opts) == 0 {
		panic("vacuum: cannot instantiate new vacuum with no policies")
	}

	policies := sliceutil.Map(opts, func(i int) policy {
		p := opts[i]

		return policy{
			every:     p.Every,
			threshold: p.IntervalSeconds(),
		}
	})

	slices.SortFunc(policies, func(a, b policy) bool {
		return a.threshold < b.threshold
	})

	thresholds := sliceutil.Map(policies, func(i int) int64 {
		return policies[i].threshold
	})

	// Find the greatest common denominator to run the pooling.
	// If the given policy interval is 3s, 6s, and 9s for example,
	// the GCD will be 3s.
	gcd := internal.GCD(thresholds)
	tick := time.Duration(gcd) * time.Second

	return &Vacuum{
		tick:     tick,
		policies: policies,
		done:     make(chan struct{}),
	}
}

func (v *Vacuum) Inc(n int64) int64 {
	return v.every.Add(n)
}

// Run executes whenever the condition is fulfilled. Returning an error will
// cause the every and timer not to reset.
// The client should be responsible for logging and handling the error.
func (v *Vacuum) Run(ctx context.Context, h Handler) func() {
	v.begin.Do(func() {
		v.wg.Add(1)

		go func() {
			defer v.wg.Done()

			v.start(ctx, h)
		}()
	})

	return v.stop
}

func (v *Vacuum) start(ctx context.Context, h Handler) {
	t := time.NewTicker(v.tick)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-v.done:
			return
		case <-t.C:
			if !v.allow() {
				continue
			}

			if err := h(ctx); err != nil {
				continue
			}

			v.reset()
		}
	}
}

func (v *Vacuum) stop() {
	v.end.Do(func() {
		close(v.done)
		v.wg.Wait()
	})
}

func (v *Vacuum) reset() {
	v.every.Store(0)
	v.unix.Store(time.Now().Unix())
}

func (v *Vacuum) allow() bool {
	delta := time.Now().Unix() - v.unix.Load()
	every := v.every.Load()

	for _, p := range v.policies {
		if delta < p.threshold {
			return false
		}

		if every >= p.every {
			return true
		}
	}

	return false
}
