package circuitbreaker_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
	"github.com/alextanhongpin/evaltest"
)

func TestCircuitBreaker(t *testing.T) {
	type Input struct {
		Status string
		N      int
	}
	type Output struct {
		Result string
		Status string
	}
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input Input) (string, error) {
		var cb *circuitbreaker.CircuitBreaker
		name := evaltest.Name(ctx)
		synctest.Test(t, func(t *testing.T) {
			cb = newCircuitbreaker()
			cb.SetStatus(circuitbreaker.ParseStatus(input.Status))
			fn := func(ctx context.Context, name string) (string, error) {
				if strings.Contains(name, "error") {
					return "", errors.ErrUnsupported
				}
				return name, nil
			}
			h := circuitbreaker.Func(fn, cb)
			for range input.N {
				res, err := h(ctx, t.Name())
				evaltest.Log(ctx, evaltest.NewT[any, any]("do", nil, &Output{Result: res, Status: cb.Status().String()}, err))
			}
			if strings.Contains(name, "sleep") {
				time.Sleep(cb.OpenTimeout)

				for range input.N {
					err := cb.Do(func() error {
						return nil
					})
					evaltest.Log(ctx, evaltest.NewT[any, any]("sleep", nil, &Output{Result: "", Status: cb.Status().String()}, err))
				}
			}
		})
		return cb.Status().String(), nil
	})
}

func newCircuitbreaker() *circuitbreaker.CircuitBreaker {
	cfg := circuitbreaker.DefaultConfig()
	cfg.FailureThreshold = 3
	cfg.SuccessThreshold = 3
	cfg.OpenTimeout = 100 * time.Millisecond

	return circuitbreaker.New(cfg)
}
