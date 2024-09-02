package clock

import (
	"cmp"
	"time"
)

const (
	eq = 0
	gt = 1
	lt = -1
)

type Bound int

const (
	Empty Bound = iota
	Unbounded
	Inclusive
	Exclusive
)

type TimeRange struct {
	Start      time.Time
	End        time.Time
	StartBound Bound
	EndBound   Bound
}

func (b *TimeRange) Overlap(now time.Time) bool {
	validStart := b.StartBound != Empty &&
		((b.StartBound == Inclusive && Gte(b.Start, now)) ||
			(b.StartBound == Exclusive && Gt(b.Start, now)) ||
			b.StartBound == Unbounded)

	validEnd := b.EndBound != Empty &&
		((b.EndBound == Inclusive && Lte(b.End, now)) ||
			(b.EndBound == Exclusive && Lt(b.End, now)) ||
			b.EndBound == Unbounded)

	return validStart && validEnd
}

func Compare(a, b time.Time) int {
	return cmp.Compare(a.UnixNano(), b.UnixNano())
}

func Gte(a, b time.Time) bool {
	c := Compare(a, b)
	return c == gt || c == eq
}

func Gt(a, b time.Time) bool {
	c := Compare(a, b)
	return c == gt
}

func Lte(a, b time.Time) bool {
	c := Compare(a, b)
	return c == lt || c == eq
}

func Lt(a, b time.Time) bool {
	c := Compare(a, b)
	return c == lt
}
