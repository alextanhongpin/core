package okay_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/types/okay"
)

var RateLimitExceeded = errors.New("rate limit exceeded")

type RateLimitKey string

type RateLimit struct {
	Remaining int
	Limit     int
	Reset     time.Duration
}

func (r *RateLimit) Unwrap() (bool, error) {
	ok := r.Remaining > 0
	if !ok {
		return false, RateLimitExceeded
	}
	return true, nil
}

func CheckRateLimit() okay.OK[RateLimitKey] {
	fn := func(ctx context.Context, key RateLimitKey) okay.Response {
		var remaining int
		if key != "0.0.0.0:banned-user" {
			remaining = 42
		}

		return &RateLimit{
			Remaining: remaining,
			Limit:     1000,
			Reset:     time.Hour,
		}
	}

	return okay.Func[RateLimitKey](fn)
}

func ExampleResponse() {
	ok := okay.New(
		CheckRateLimit(),
	)

	ctx := context.Background()
	res := ok.All(ctx, RateLimitKey("0.0.0.0:banned-user"))
	valid, err := res.Unwrap()

	fmt.Println("OK:", valid)
	fmt.Println("ERR:", err)

	fmt.Println("IsRateLimitExceeded:", errors.Is(err, RateLimitExceeded))
	rateLimit, _ := res.(*RateLimit)
	fmt.Printf("RateLimit: %+v\n", rateLimit)

	valid, err = ok.All(context.Background(), "0.0.0.0:good-user").Unwrap()
	fmt.Println("OK:", valid)
	fmt.Println("ERR:", err)

	// Output:
	// OK: false
	// ERR: rate limit exceeded
	// IsRateLimitExceeded: true
	// RateLimit: &{Remaining:0 Limit:1000 Reset:1h0m0s}
	// OK: true
	// ERR: <nil>
}
