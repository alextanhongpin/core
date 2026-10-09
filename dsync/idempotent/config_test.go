package idempotent

import (
	"context"
	"testing"
	"time"
)

func TestHandlerValueDefaultsAndValidation(t *testing.T) {
	i := MustNew(&failingRenewal{})
	fn := HandlerFunc[int, int](func(context.Context, int) (int, error) { return 0, nil })
	if _, err := i.HandlerFunc(fn, HandlerConfig{}); err != nil {
		t.Fatal(err)
	}
	for _, cfg := range []HandlerConfig{{LockTTL: -time.Second}, {KeepTTL: -time.Second}, {LockTTL: time.Nanosecond}} {
		if _, err := i.HandlerFunc(fn, cfg); err == nil {
			t.Fatal("invalid config accepted")
		}
	}
	if _, err := New(nil); err == nil {
		t.Fatal("nil client accepted")
	}
	if _, err := i.HandlerFunc[int, int](nil, HandlerConfig{}); err == nil {
		t.Fatal("nil task accepted")
	}
}
