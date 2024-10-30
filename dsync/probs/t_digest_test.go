package probs_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/dsync/probs"
	"github.com/alextanhongpin/core/storage/redis/redistest"
	"github.com/stretchr/testify/assert"
)

func TestTDigest(t *testing.T) {
	td := probs.NewTDigest(redistest.Client(t), 100)

	ms := func(d time.Duration) float64 {
		return float64(d.Milliseconds())
	}

	// Measure API latency.
	key := t.Name() + ":t_digest:GET /healthz"
	is := assert.New(t)
	is.Nil(td.CreateAndAdd(ctx, key,
		ms(10*time.Millisecond),
		ms(100*time.Millisecond),
		ms(1000*time.Millisecond),
	))
}
