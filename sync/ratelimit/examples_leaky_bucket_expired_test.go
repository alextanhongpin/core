package ratelimit_test

import (
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func ExampleLeakyBucketExpired() {
	rl := ratelimit.NewLeakyBucket(5, time.Second, 0)

	now := time.Now()
	periods := []time.Duration{
		0,
		// Bursts at the end
		600 * time.Millisecond,
		601 * time.Millisecond,

		799 * time.Millisecond,
		800 * time.Millisecond,
		801 * time.Millisecond,

		999 * time.Millisecond,
		1000 * time.Millisecond,
		1001 * time.Millisecond,
	}

	for _, p := range periods {
		rl.Now = func() time.Time {
			return now.Add(p)
		}
		allow := p.Milliseconds()%2 == 0

		checkAllow := rl.AllowAt(now.Add(p), 1)
		result := rl.Allow()
		if result.Allow != allow || result.Allow != checkAllow {
			panic("doesn't allow")
		}

		fmt.Println(now.Add(p).Sub(now), result.Allow, result.Remaining)
	}
	// Output:
	// 0s true 4
	// 600ms true 1
	// 601ms false 1
	// 799ms false 1
	// 800ms true 0
	// 801ms false 0
	// 999ms false 0
	// 1s true 4
	// 1.001s false 4
}
