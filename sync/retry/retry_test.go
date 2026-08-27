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

		rt := newRetry()
		rt.Attempts = input
		rt.Backoff = retry.NewConstantBackoff(1 * time.Millisecond)
		if strings.Contains(name, "noretry") {
			rt.Retryable = retry.NonRetryableErrors(errors.ErrUnsupported)
		}
		h := retry.Func(fn, rt)
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

func TestThrottle(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input int) (*DoOutput, error) {
		throttler := retry.NewThrottler(&retry.ThrottlerConfig{
			MaxTokens:  2,
			TokenRatio: 0.1,
		})

		rt := newRetry()
		rt.Throttler = throttler
		rt.Attempts = input

		var count int

		h := retry.Func(func(ctx context.Context, input any) (any, error) {
			count++
			return nil, errors.ErrUnsupported
		}, rt)
		_, err := h(ctx, nil)

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

func newRetry() *retry.Retry {
	cfg := retry.DefaultConfig()
	cfg.Attempts = 10
	cfg.Backoff = retry.NewConstantBackoff(0)
	cfg.Throttler = retry.NewNoopThrottler()
	rt := retry.New(cfg)
	return rt
}
