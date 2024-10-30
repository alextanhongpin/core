package probs_test

import (
	"context"
	"testing"

	"github.com/alextanhongpin/core/storage/redis/redistest"
)

var ctx = context.Background()

func TestMain(m *testing.M) {
	stop := redistest.Init()
	defer stop()

	m.Run()
}
