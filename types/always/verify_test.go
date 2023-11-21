package always_test

import (
	"context"
	"errors"
	"testing"

	"github.com/alextanhongpin/core/types/always"
)

type mockVerifier[T any] struct {
	verifyFunc func(context.Context, T) error
}

func (m *mockVerifier[T]) Verify(ctx context.Context, t T) error {
	return m.verifyFunc(ctx, t)
}

func TestVerify(t *testing.T) {
	t.Run("all verifiers pass", func(t *testing.T) {
		v1 := &mockVerifier[int]{
			verifyFunc: func(ctx context.Context, t int) error {
				return nil
			},
		}
		v2 := &mockVerifier[int]{
			verifyFunc: func(ctx context.Context, t int) error {
				return nil
			},
		}
		v3 := &mockVerifier[int]{
			verifyFunc: func(ctx context.Context, t int) error {
				return nil
			},
		}

		err := always.Verify(context.Background(), 1, v1.Verify, v2.Verify, v3.Verify)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("one verifier fails", func(t *testing.T) {
		v1 := &mockVerifier[int]{
			verifyFunc: func(ctx context.Context, t int) error {
				return nil
			},
		}
		v2 := &mockVerifier[int]{
			verifyFunc: func(ctx context.Context, t int) error {
				return errors.New("verification failed")
			},
		}
		v3 := &mockVerifier[int]{
			verifyFunc: func(ctx context.Context, t int) error {
				return nil
			},
		}

		err := always.Verify(context.Background(), 1, v1.Verify, v2.Verify, v3.Verify)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("multiple verifiers fail", func(t *testing.T) {
		v1 := &mockVerifier[int]{
			verifyFunc: func(ctx context.Context, t int) error {
				return errors.New("verification failed")
			},
		}
		v2 := &mockVerifier[int]{
			verifyFunc: func(ctx context.Context, t int) error {
				return errors.New("verification failed")
			},
		}
		v3 := &mockVerifier[int]{
			verifyFunc: func(ctx context.Context, t int) error {
				return errors.New("verification failed")
			},
		}

		err := always.Verify(context.Background(), 1, v1.Verify, v2.Verify, v3.Verify)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("no verifiers", func(t *testing.T) {
		err := always.Verify(context.Background(), 1)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}
