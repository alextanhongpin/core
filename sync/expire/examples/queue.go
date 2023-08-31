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
	worker := expire.NewQueue(expire.QueueOption{
		Window: 5 * time.Second,
		Handler: func() {
			fmt.Println("run...", time.Now())
		},
	})
	defer worker.Stop()

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt)
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(1000 * time.Millisecond)

				sleep := time.Duration(rand.Intn(10)) * time.Second
				fmt.Println("task:adding", sleep)
				worker.Add(ctx, time.Now().Add(sleep))
			}
		}
	}()

	fmt.Println("running...")
	<-ctx.Done()

	fmt.Println("terminating...")
}
