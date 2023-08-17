package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alextanhongpin/core/sync/snapshot"
)

func main() {
	snap := snapshot.New([]snapshot.Policy{
		{Every: 10_000, Interval: 5 * time.Second},
		{Every: 1_000, Interval: 10 * time.Second},
		{Every: 100, Interval: 20 * time.Second},
		{Every: 10, Interval: 30 * time.Second},
		{Every: 1, Interval: 1 * time.Minute},
	})

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	snap.Inc(10_000)

	stop := snap.Run(ctx, func(ctx context.Context) {
		fmt.Println("run")
	})

	defer stop()

	<-ctx.Done()

	fmt.Println("terminating ...")
}
