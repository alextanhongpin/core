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
var ErrIntervalRangeInvalid = errors.New("vacuum: interval range must be in seconds")

type Policy struct {
	Count    int64
	Interval time.Duration
}

func (p Policy) Valid() error {
	ns := p.Interval.Nanoseconds()
	if ns%1e9 > 0 {
		return ErrIntervalRangeInvalid
	}

	return nil
}

func (p Policy) IntervalSeconds() int64 {
	return p.Interval.Nanoseconds() / 1e9
}

type policy struct {
	count     int64
	threshold int64
}

type Handler func(ctx context.Context) error

type Vacuum struct {
	unix     atomic.Int64
	policies []policy
	tick     time.Duration
	count    atomic.Int64
	doneCh   chan struct{}
	errCh    chan error
	begin    sync.Once
	end      sync.Once
}

func NewPolicy(count int64, interval time.Duration) Policy {
	return Policy{
		Count:    count,
		Interval: interval,
	}
}

func New(opts []Policy) *Vacuum {
	if len(opts) == 0 {
		panic("vacuum: cannot instantiate new vacuum with no policies")
	}

	policies, err := sliceutil.MapError(opts, func(i int) (policy, error) {
		p := opts[i]

		if err := p.Valid(); err != nil {
			return policy{}, err
		}

		return policy{
			count:     p.Count,
			threshold: p.IntervalSeconds(),
		}, nil
	})
	if err != nil {
		panic(err)
	}

	slices.SortFunc(policies, func(a, b policy) bool {
		return a.threshold < b.threshold
	})

	periods := sliceutil.Map(policies, func(i int) int64 {
		return policies[i].threshold
	})

	// Find the greatest common denominator to run the pooling.
	// If the given policy interval is 3s, 6s, and 9s for example,
	// the GCD will be 3s.
	gcd := internal.GCD(periods)
	tick := time.Duration(gcd) * time.Second

	return &Vacuum{
		tick:     tick,
		policies: policies,
		doneCh:   make(chan struct{}),
		errCh:    make(chan error, 1),
	}
}

func (v *Vacuum) Inc(n int64) int64 {
	return v.count.Add(n)
}

// Run executes whenever the condition is fulfilled. Returning an error will
// cause the count and timer not to reset.
// The client should be responsible for logging and handling the error.
func (v *Vacuum) Run(ctx context.Context, h Handler) func() error {
	v.begin.Do(func() {
		go func() {
			for {
				err := v.start(ctx, h)
				if errors.Is(err, ErrClosed) {
					v.errCh <- err
					close(v.errCh)
					return
				}
			}
		}()

	})

	return v.stop
}

func (v *Vacuum) start(ctx context.Context, h Handler) error {
	t := time.NewTicker(v.tick)
	defer t.Stop()

	for {
		select {
		case <-v.doneCh:
			return ErrClosed
		case <-t.C:
			if v.allow() {
				if err := h(ctx); err != nil {
					return err
				}

				v.reset()
			}
		}
	}
}

func (v *Vacuum) stop() (err error) {
	v.end.Do(func() {
		close(v.doneCh)
		err = <-v.errCh
	})

	return
}

func (v *Vacuum) reset() {
	v.count.Store(0)
	v.unix.Store(time.Now().Unix())
}

func (v *Vacuum) allow() bool {
	delta := time.Now().Unix() - v.unix.Load()
	count := v.count.Load()

	for _, p := range v.policies {
		if delta < p.threshold {
			return false
		}

		if count >= p.count {
			return true
		}
	}

	return false
}
