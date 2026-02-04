package ratelimit

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"
)

var (
	ErrInvalidNumber = errors.New("ratelimit: value must be positive")
)

type option struct {
	limit  int
	period time.Duration
	burst  int
}

func (o *option) Validate() error {
	if o.limit <= 0 {
		return fmt.Errorf("%w: limit", ErrInvalidNumber)
	}
	if o.period <= 0 {
		return fmt.Errorf("%w: period", ErrInvalidNumber)
	}
	if o.burst < 0 {
		return fmt.Errorf("%w: burst", ErrInvalidNumber)
	}
	return nil
}

var (
	maxInt64 = big.NewInt(math.MaxInt64)
	minInt64 = big.NewInt(math.MinInt64)
)

func clamp(v *big.Int) *big.Int {
	if v.Cmp(maxInt64) > 0 {
		v.Set(maxInt64)
	}
	if v.Cmp(minInt64) < 0 {
		v.Set(minInt64)
	}
	return v
}

func add(a, b int64) int64 {
	m := big.NewInt(a)
	n := big.NewInt(b)
	r := new(big.Int).Add(m, n)
	return clamp(r).Int64()
}

func mul(a, b int64) int64 {
	m := big.NewInt(a)
	n := big.NewInt(b)
	r := new(big.Int).Mul(m, n)
	return clamp(r).Int64()
}

func div(a, b int64) int64 {
	m := big.NewInt(a)
	n := big.NewInt(b)
	r := new(big.Int).Div(m, n)
	return clamp(r).Int64()
}
