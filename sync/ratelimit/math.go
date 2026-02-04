package ratelimit

import (
	"errors"
	"math"
	"math/big"
)

var (
	ErrInvalidNumber = errors.New("ratelimit: value must be positive")
)

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
