package always_test

import (
	"errors"
	"testing"

	"github.com/alextanhongpin/core/types/always"
)

type mockValidatable struct {
	valid bool
}

func (m *mockValidatable) Valid() error {
	if !m.valid {
		return errors.New("validation failed")
	}
	return nil
}

func TestValid(t *testing.T) {
	t.Run("all valid", func(t *testing.T) {
		v1 := &mockValidatable{valid: true}
		v2 := &mockValidatable{valid: true}
		v3 := &mockValidatable{valid: true}

		err := always.Valid(v1, v2, v3)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("one invalid", func(t *testing.T) {
		v1 := &mockValidatable{valid: true}
		v2 := &mockValidatable{valid: false}
		v3 := &mockValidatable{valid: true}

		err := always.Valid(v1, v2, v3)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("multiple invalid", func(t *testing.T) {
		v1 := &mockValidatable{valid: false}
		v2 := &mockValidatable{valid: false}
		v3 := &mockValidatable{valid: false}

		err := always.Valid(v1, v2, v3)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("valid func", func(t *testing.T) {
		fn := always.ValidFunc(func() error {
			return nil
		})
		if err := fn.Valid(); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}
