package retry_test

import (
	"fmt"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
)

func ExampleNewBackoff() {
	fmt.Println("constant:", retry.NewBackoff(retry.BackoffTypeConstant, 10, 1*time.Second))
	fmt.Println("linear:", retry.NewBackoff(retry.BackoffTypeLinear, 10, 1*time.Second))
	fmt.Println("exponential:", retry.NewBackoff(retry.BackoffTypeExponential, 10, 100*time.Millisecond))
	// Output:
	// constant: [1s 1s 1s 1s 1s 1s 1s 1s 1s 1s]
	// linear: [1s 2s 3s 4s 5s 6s 7s 8s 9s 10s]
	// exponential: [100ms 200ms 400ms 800ms 1.6s 3.2s 6.4s 12.8s 25.6s 51.2s]
}
