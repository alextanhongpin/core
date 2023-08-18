package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alextanhongpin/core/sync/expire"
)

func main() {
	worker := expire.New(expire.Option{
		Threshold: 10,
		Interval:  5 * time.Second,
	})

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt)
	defer cancel()

	stop := worker.Run(ctx, func(ctx context.Context) error {
		fmt.Println("executing ...", time.Now())
		return nil
	})
	defer stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				sleep := time.Duration(rand.Intn(1_000)) * time.Millisecond
				time.Sleep(sleep)

				fmt.Println("added 1s")
				worker.Add(time.Now().Add(1 * time.Second))
			}
		}
	}()

	fmt.Println("running...")
	<-ctx.Done()

	fmt.Println("terminating...")
}