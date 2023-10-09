package ratelimit_test

import (
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func ExampleLeakyBucket() {
	rl := ratelimit.NewLeakyBucket(5, time.Second, 0)

	now := time.Now()
	periods := []time.Duration{
		0,
		199 * time.Millisecond,
		200 * time.Millisecond,
		201 * time.Millisecond,

		399 * time.Millisecond,
		400 * time.Millisecond,
		401 * time.Millisecond,

		599 * time.Millisecond,
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

		dryRun := rl.AllowAt(now.Add(p), 1)
		result := rl.Allow()
		if result.Allow != allow || result.Allow != dryRun {
			panic("doesn't allow")
		}

		fmt.Println(now.Add(p).Sub(now), result.Allow, result.Remaining)
	}
	// Output:
	// 0s true 4
	// 199ms false 4
	// 200ms true 3
	// 201ms false 3
	// 399ms false 3
	// 400ms true 2
	// 401ms false 2
	// 599ms false 2
	// 600ms true 1
	// 601ms false 1
	// 799ms false 1
	// 800ms true 0
	// 801ms false 0
	// 999ms false 0
	// 1s true 4
	// 1.001s false 4
}
