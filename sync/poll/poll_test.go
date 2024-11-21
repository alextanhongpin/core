package poll_test

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/alextanhongpin/core/sync/poll"
)

func TestPoll(t *testing.T) {
	p := poll.New()
	p.Logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	var wg sync.WaitGroup
	wg.Add(1)
	stop := p.Poll(func(ctx context.Context) error {
		defer wg.Done()
		return poll.EOQ
	})

	wg.Wait()
	stop()
}
