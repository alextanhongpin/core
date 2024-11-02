package random

import (
	"math/rand/v2"
	"time"
)

func Duration(duration time.Duration) time.Duration {
	return time.Duration(rand.Int64N(duration.Milliseconds())) * time.Millisecond
}

func DurationBetween(lo, hi time.Duration) time.Duration {
	return Duration(hi-lo) + lo
}
