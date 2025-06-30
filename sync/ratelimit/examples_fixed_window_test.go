package ratelimit_test

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func ExampleFixedWindow() {
	now := time.Now()

	rl := ratelimit.MustNewFixedWindow(5, time.Second)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t\n", "t", "allow")
	call := func(duration time.Duration, allow bool) {
		rl.Now = func() time.Time {
			return now.Add(duration)
		}

		allowed := rl.Allow()
		if allow != allowed {
			log.Fatalf("unexpected allow at %v", duration)
		}
		fmt.Fprintf(w, "%s\t%t\t\n", duration, allowed)
	}

	call(0, true)
	call(1*time.Millisecond, true)
	call(2*time.Millisecond, true)
	call(3*time.Millisecond, true)
	call(4*time.Millisecond, true)
	call(99*time.Millisecond, false)
	call(100*time.Millisecond, false)
	call(101*time.Millisecond, false)
	call(999*time.Millisecond, false)
	call(1000*time.Millisecond, true)
	call(1100*time.Millisecond, true)
	call(1200*time.Millisecond, true)
	call(1300*time.Millisecond, true)
	call(1400*time.Millisecond, true)
	call(1500*time.Millisecond, false)
	call(1999*time.Millisecond, false)

	fmt.Println("fixed window")
	w.Flush()

	// Output:
	// fixed window
	//       t| allow|
	//      0s|  true|
	//     1ms|  true|
	//     2ms|  true|
	//     3ms|  true|
	//     4ms|  true|
	//    99ms| false|
	//   100ms| false|
	//   101ms| false|
	//   999ms| false|
	//      1s|  true|
	//    1.1s|  true|
	//    1.2s|  true|
	//    1.3s|  true|
	//    1.4s|  true|
	//    1.5s| false|
	//  1.999s| false|
}
