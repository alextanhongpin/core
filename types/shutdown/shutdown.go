package shutdown

import (
	"context"
	"errors"
	"sync"
)

// Shutdown is a list of cleanup callbacks. The zero value is ready to use.
type Shutdown []func(context.Context) error

// Append registers fn. Nil functions are ignored.
func (s *Shutdown) Append(fn func(context.Context) error) {
	if fn == nil {
		return
	}
	*s = append(*s, fn)
}

// Wait runs every callback concurrently with ctx and blocks until all of them
// return, even after ctx is done. It returns the non-nil errors joined with
// errors.Join, in registration order. Errors matching ctx.Err() are dropped.
func (s Shutdown) Wait(ctx context.Context) error {
	errs := make([]error, len(s))
	var wg sync.WaitGroup
	for i, fn := range s {
		wg.Go(func() {
			errs[i] = fn(ctx)
		})
	}
	wg.Wait()

	if ctxErr := ctx.Err(); ctxErr != nil {
		for i, err := range errs {
			if errors.Is(err, ctxErr) {
				errs[i] = nil
			}
		}
	}
	return errors.Join(errs...)
}
