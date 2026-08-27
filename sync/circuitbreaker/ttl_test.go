package circuitbreaker_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/circuitbreaker"
	"github.com/alextanhongpin/evaltest"
)

func TestTTL(t *testing.T) {
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input int) (any, error) {
		ttl := &circuitbreaker.TTL{
			Value:     input,
			ExpiresAt: time.Now().Add(time.Second),
		}
		name := evaltest.Name(ctx)
		if strings.Contains(name, "expires") {
			ttl.ExpiresAt = time.Now().Add(-time.Second)
		}

		return ttl.Load(), nil
	})
}
