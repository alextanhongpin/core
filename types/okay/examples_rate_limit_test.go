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

func (r *RateLimit) OK() bool {
	return r.Remaining > 0
}

func (r *RateLimit) Err() error {
	if r.OK() {
		return nil
	}

	return RateLimitExceeded
}

func CheckRateLimit() okay.OK[RateLimitKey] {
	fn := func(ctx context.Context, key RateLimitKey) okay.Response {
		if key == "0.0.0.0:banned-user" {
			return &RateLimit{
				Limit: 1000,
				Reset: time.Hour,
			}
		}

		return &RateLimit{
			Remaining: 42,
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
	res := okay.Check[RateLimitKey](ctx, RateLimitKey("0.0.0.0:banned-user"), ok)

	fmt.Println("OK:", res.OK())
	fmt.Println("ERR:", res.Err())

	fmt.Println("IsRateLimitExceeded:", errors.Is(res.Err(), RateLimitExceeded))
	rateLimit, _ := res.(*RateLimit)
	fmt.Printf("RateLimit: %+v\n", rateLimit)

	res = ok.Allows(context.Background(), "0.0.0.0:good-user")
	fmt.Println("OK:", res.OK())
	fmt.Println("ERR:", res.Err())

	// Output:
	// OK: false
	// ERR: rate limit exceeded
	// IsRateLimitExceeded: true
	// RateLimit: &{Remaining:0 Limit:1000 Reset:1h0m0s}
	// OK: true
	// ERR: <nil>
}
