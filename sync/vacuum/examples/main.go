package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alextanhongpin/core/sync/vacuum"
)

func main() {
	vac := vacuum.New(vacuum.Option{
		Policies: []vacuum.Policy{
			{Count: 10_000, Interval: 5 * time.Second},
			{Count: 1_000, Interval: 10 * time.Second},
			{Count: 100, Interval: 20 * time.Second},
			{Count: 10, Interval: 30 * time.Second},
			{Count: 1, Interval: 1 * time.Minute},
		},
	})

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	vac.Inc(10_000)

	stop := vac.Run(ctx, func(ctx context.Context) error {
		fmt.Println("run")
		return errors.New("intended")
	})

	defer func() {
		if err := stop(); err != nil && !errors.Is(err, vacuum.ErrClosed) {
			log.Fatal(err)
		}

		fmt.Println("terminated")
	}()

	<-ctx.Done()

	fmt.Println("terminating ...")
}
