package retry

import (
	"cmp"
	"sync"
)

type throttler interface {
	Allow() bool
	Success()
}

var _ throttler = (*Throttler)(nil)

type Throttler struct {
	ratio  float64
	thresh float64 // max / 2
	max    float64

	mu     sync.Mutex
	tokens float64
}

type ThrottlerOptions struct {
	MaxTokens  float64
	TokenRatio float64
}

func NewThrottlerOptions() *ThrottlerOptions {
	return &ThrottlerOptions{
		MaxTokens:  10,
		TokenRatio: 0.1,
	}
}

func NewThrottler(opts *ThrottlerOptions) *Throttler {
	opts = cmp.Or(opts, NewThrottlerOptions())

	ratio := opts.TokenRatio
	maxTokens := opts.MaxTokens

	return &Throttler{
		ratio:  ratio,
		max:    maxTokens,
		tokens: maxTokens,
		thresh: maxTokens / 2,
	}
}

func (t *Throttler) Allow() bool {
	if t == nil {
		return true
	}

	t.mu.Lock()
	t.tokens = max(t.tokens-1, 0)
	ok := t.tokens > t.thresh
	t.mu.Unlock()

	return ok
}

func (t *Throttler) Success() {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.tokens = min(t.tokens+t.ratio, t.max)
	t.mu.Unlock()
}
