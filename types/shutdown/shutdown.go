package shutdown

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

type Shutdown []func(context.Context) error

func (s *Shutdown) Append(fn func(context.Context) error) {
	*s = append(*s, fn)
}

func (s Shutdown) Wait(ctx context.Context) {
	var wg sync.WaitGroup
	for i := len(s) - 1; i > -1; i-- {
		wg.Go(func() {
			err := s[i](ctx)
			if err != nil && !errors.Is(err, ctx.Err()) {
				slog.Error("shutdown error", "err", err.Error())
			}
		})
	}

	wg.Wait()
}
