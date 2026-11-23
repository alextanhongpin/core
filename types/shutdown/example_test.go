package shutdown_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alextanhongpin/core/types/shutdown"
)

func ExampleShutdown() {
	var s shutdown.Shutdown
	s.Append(func(ctx context.Context) error { return nil }) // e.g. server.Shutdown
	s.Append(func(ctx context.Context) error { return errors.New("db: close failed") })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fmt.Println(s.Wait(ctx))
	// Output: db: close failed
}
