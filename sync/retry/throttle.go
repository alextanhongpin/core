package retry

import (
	"cmp"
	"sync"
)

func NewNoopThrottler() *NoopThrottler {
	return &NoopThrottler{}
}

type NoopThrottler struct{}

func (n *NoopThrottler) Allow() bool {
	return true
}
func (n *NoopThrottler) Success() {}

type Limiter interface {
	Allow() bool
	Success()
}

var _ Limiter = (*Throttler)(nil)
var _ Limiter = (*NoopThrottler)(nil)

type Throttler struct {
	ratio  float64
	thresh float64 // max / 2
	max    float64

	mu     sync.Mutex
	tokens float64
}

type ThrottlerConfig struct {
	MaxTokens  float64
	TokenRatio float64
}

func DefaultThrottlerConfig() *ThrottlerConfig {
	return &ThrottlerConfig{
		MaxTokens:  10,
		TokenRatio: 0.1,
	}
}

func NewThrottler(cfg *ThrottlerConfig) *Throttler {
	cfg = cmp.Or(cfg, DefaultThrottlerConfig())

	ratio := cfg.TokenRatio
	maxTokens := cfg.MaxTokens

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
	defer t.mu.Unlock()

	if t.tokens <= t.thresh {
		return false
	}

	t.tokens = max(t.tokens-1, 0)
	return true
}

func (t *Throttler) Success() {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.tokens = min(t.tokens+t.ratio, t.max)
	t.mu.Unlock()
}
