package futures_test

import (
	"context"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/exp/futures"
)

func ExampleFunc() {
	f0 := futures.Func[int](func(ctx context.Context) (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 42, nil
	})
	f1 := futures.Func[int](func(ctx context.Context) (int, error) {
		time.Sleep(100 * time.Millisecond)
		return 3173, nil
	})

	now := time.Now()
	res := futures.Join(context.Background(), f0, f1)
	isWithinSLA := time.Since(now) < 110*time.Millisecond

	meaningOfLife, err := res[0].Unwrap()
	if err != nil {
		panic(err)
	}

	elie, err := res[1].Unwrap()
	if err != nil {
		panic(err)
	}

	fmt.Println(isWithinSLA)
	fmt.Println(meaningOfLife)
	fmt.Println(elie)
	// Output
	// true
	// 42
	// 3173
}
