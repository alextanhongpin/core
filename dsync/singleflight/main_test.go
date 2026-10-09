package singleflight_test

import (
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	stop := redistest.Init(redistest.Options{Image: "redis:8.6.2"})
	code := m.Run()
	stop()
	os.Exit(code)
}
