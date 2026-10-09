package ratelimit

import (
	"errors"
	"math"
	"time"
)

type Config struct {
	Limit  int
	Period time.Duration
	Burst  int
}

func DefaultConfig() Config { return Config{Limit: 100, Period: time.Minute} }
func (c Config) WithDefaults() Config {
	if c.Limit == 0 {
		c.Limit = 100
	}
	if c.Period == 0 {
		c.Period = time.Minute
	}
	return c
}
func (c Config) Validate() error {
	if c.Limit <= 0 || c.Burst < 0 || float64(c.Limit) > 9007199254740991 || float64(c.Burst) > 9007199254740990 {
		return errors.New("ratelimit: invalid limit or burst")
	}
	if c.Period < time.Millisecond {
		return errors.New("ratelimit: period must be at least one millisecond")
	}
	return nil
}
func validateRequest(key string, n int) error {
	if n < 0 {
		return ErrNegative
	}
	if key == "" || float64(n) > 9007199254740991 {
		return errors.New("ratelimit: invalid key or quantity")
	}
	return nil
}
func (c Config) validateGCRA() error {
	if c.Period.Milliseconds()/int64(c.Limit) < 1 {
		return errors.New("ratelimit: emission interval must be at least one millisecond")
	}
	if float64(c.Burst+1)*float64(c.Period.Milliseconds())/float64(c.Limit) > float64(math.MaxInt64/int64(time.Millisecond)) {
		return errors.New("ratelimit: burst duration overflows")
	}
	return nil
}
