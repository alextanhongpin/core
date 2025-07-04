package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func BenchmarkBreaker(b *testing.B) {
	b.Run("successful_requests", func(b *testing.B) {
		cb := New()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cb.Do(func() error {
					return nil
				})
			}
		})
	})

	b.Run("failed_requests", func(b *testing.B) {
		cb := New()
		testErr := errors.New("test error")
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cb.Do(func() error {
					return testErr
				})
			}
		})
	})

	b.Run("circuit_open", func(b *testing.B) {
		cb := NewWithOptions(Options{
			FailureThreshold: 1,
			FailureRatio:     0.1,
		})

		// Force circuit to open
		cb.Do(func() error { return errors.New("force open") })
		cb.Do(func() error { return errors.New("force open") })

		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cb.Do(func() error {
					return nil
				})
			}
		})
	})

	b.Run("with_callbacks", func(b *testing.B) {
		cb := NewWithOptions(Options{
			OnRequest: func() {},
			OnSuccess: func(duration time.Duration) {},
			OnFailure: func(err error, duration time.Duration) {},
		})
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cb.Do(func() error {
					return nil
				})
			}
		})
	})

	b.Run("metrics_access", func(b *testing.B) {
		cb := New()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = cb.Metrics()
			}
		})
	})
}

func BenchmarkBreakervsPlainCall(b *testing.B) {
	cb := New()

	b.Run("plain_call", func(b *testing.B) {
		fn := func() error { return nil }
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			fn()
		}
	})

	b.Run("circuit_breaker", func(b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			cb.Do(func() error {
				return nil
			})
		}
	})
}

func BenchmarkBreakerStateCheck(b *testing.B) {
	cb := New()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.Status()
		}
	})
}
