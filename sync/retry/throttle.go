package retry

import (
	"fmt"
	"math"
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

// Limiter accounts for retries and successful logical operations. Implementations
// must support concurrent calls when shared. Allow consumes a token on admission.
type Limiter interface {
	Allow() bool
	Success()
}

var _ Limiter = (*Throttler)(nil)
var _ Limiter = (*NoopThrottler)(nil)

// Throttler is a concurrency-safe adaptive budget. It does not refill over time.
// Only Success replenishes tokens; initial attempts remain allowed by Retry.
type Throttler struct {
	ratio  float64
	thresh float64 // max / 2
	max    float64

	mu     sync.Mutex
	tokens float64
}

type ThrottlerConfig struct {
	// MaxTokens defaults to ten when zero.
	MaxTokens float64
	// TokenRatio is the refill per success; zero disables replenishment.
	TokenRatio float64
}

func DefaultThrottlerConfig() ThrottlerConfig {
	return ThrottlerConfig{
		MaxTokens:  10,
		TokenRatio: 0.1,
	}
}

// WithDefaults fills the capacity without changing the explicit refill ratio.
func (c ThrottlerConfig) WithDefaults() ThrottlerConfig {
	if c.MaxTokens == 0 {
		c.MaxTokens = 10
	}
	return c
}

// Validate rejects negative or non-finite budget settings.
func (c ThrottlerConfig) Validate() error {
	if c.MaxTokens <= 0 || math.IsNaN(c.MaxTokens) || math.IsInf(c.MaxTokens, 0) {
		return fmt.Errorf("retry: max tokens must be finite and positive")
	}
	if c.TokenRatio < 0 || math.IsNaN(c.TokenRatio) || math.IsInf(c.TokenRatio, 0) {
		return fmt.Errorf("retry: token ratio must be finite and nonnegative")
	}
	return nil
}

// NewThrottler constructs an adaptive budget from a configuration copy.
func NewThrottler(cfg ThrottlerConfig) (*Throttler, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ratio := cfg.TokenRatio
	maxTokens := cfg.MaxTokens

	return &Throttler{
		ratio:  ratio,
		max:    maxTokens,
		tokens: maxTokens,
		thresh: maxTokens / 2,
	}, nil
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
