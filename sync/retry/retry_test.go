package retry_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/retry"
	"github.com/alextanhongpin/evaltest"
)

func TestHandler(t *testing.T) {
	type Output struct {
		Attempts int
		Result   string
	}
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input int) (*Output, error) {
		name := evaltest.Name(ctx)

		var count int
		fn := func(ctx context.Context, input string) (string, error) {
			count++
			if ctx.Err() != nil {
				return "", context.Cause(ctx)
			}
			if strings.Contains(name, "error") {
				return "", errors.ErrUnsupported
			}
			return t.Name(), nil
		}

		opts := []retry.Option{
			retry.N(input),
			retry.Constant(1 * time.Millisecond),
		}
		if strings.Contains(name, "noretry") {
			opts = append(opts, retry.WithNonRetryableErrors(errors.ErrUnsupported))
		}
		h := retry.Handler(fn, opts...)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if strings.Contains(name, "cancel") {
			cancel()
		}

		res, err := h(ctx, t.Name())
		return &Output{
			Attempts: count,
			Result:   res,
		}, err
	})
}
