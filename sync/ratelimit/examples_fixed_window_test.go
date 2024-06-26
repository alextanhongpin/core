package ratelimit_test

import (
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func ExampleFixedWindow() {
	rl := ratelimit.NewFixedWindow(5, time.Second)

	now := time.Now()
	periods := []time.Duration{
		0,
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
		99 * time.Millisecond,
		100 * time.Millisecond,
		101 * time.Millisecond,
		999 * time.Millisecond,
		1000 * time.Millisecond,
		1001 * time.Millisecond,
	}

	for _, p := range periods {
		rl.Now = func() time.Time {
			return now.Add(p)
		}
		allow := p.Milliseconds() < 5 || p >= 1000*time.Millisecond

		dryRun := rl.AllowAt(now.Add(p), 1)
		result := rl.Allow()
		if result.Allow != allow || result.Allow != dryRun {
			panic("doesn't allow")
		}

		fmt.Println(now.Add(p).Sub(now), result.Allow, result.Remaining)
	}
	// Output:
	// 0s true 4
	// 1ms true 3
	// 2ms true 2
	// 3ms true 1
	// 4ms true 0
	// 99ms false 0
	// 100ms false 0
	// 101ms false 0
	// 999ms false 0
	// 1s true 4
	// 1.001s true 3
}
