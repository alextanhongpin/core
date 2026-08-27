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

type Output struct {
	Attempts int
	Result   string
}

type DoOutput struct {
	Attempts int
}

func TestHandler(t *testing.T) {
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

func TestDo(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input int) (*DoOutput, error) {
		name := evaltest.Name(ctx)
		var count int
		err := retry.Do(ctx, func(ctx context.Context) error {
			count++
			if strings.Contains(name, "error") {
				return errors.ErrUnsupported
			}
			if strings.Contains(name, "retry") && count < 2 {
				return errors.ErrUnsupported
			}
			return nil
		}, retry.N(input), retry.NoWait)

		return &DoOutput{
			Attempts: count,
		}, err
	})
}

func TestDoValue(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input int) (*Output, error) {
		name := evaltest.Name(ctx)
		var count int
		res, err := retry.DoValue(ctx, func(ctx context.Context) (string, error) {
			count++
			if strings.Contains(name, "error") {
				return "", errors.ErrUnsupported
			}
			if strings.Contains(name, "retry") && count < 2 {
				return "", errors.ErrUnsupported
			}
			return "ok", nil
		}, retry.N(input), retry.NoWait)

		return &Output{
			Attempts: count,
			Result:   res,
		}, err
	})
}

func TestThrottle(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input int) (*DoOutput, error) {
		throttler := retry.NewThrottler(&retry.ThrottlerOptions{
			MaxTokens:  2,
			TokenRatio: 0.1,
		})
		var count int
		err := retry.Do(ctx, func(ctx context.Context) error {
			count++
			return errors.ErrUnsupported
		}, retry.N(input), retry.NoWait, retry.WithThrottler(throttler))

		return &DoOutput{
			Attempts: count,
		}, err
	})
}

func TestBackoff(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input int) (int64, error) {
		name := evaltest.Name(ctx)
		switch {
		case strings.Contains(name, "constant"):
			b := retry.NewConstantBackoff(10 * time.Millisecond)
			return int64(b.At(input)), nil
		case strings.Contains(name, "linear"):
			b := retry.NewLinearBackoff(10 * time.Millisecond)
			return int64(b.At(input)), nil
		default:
			return 0, nil
		}
	})
}
