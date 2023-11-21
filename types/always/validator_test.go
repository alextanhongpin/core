package always_test

import (
	"errors"
	"testing"

	"github.com/alextanhongpin/core/types/always"
)

type mockValidator[T any] struct {
	valid bool
}

func (m *mockValidator[T]) Validate(t T) error {
	if !m.valid {
		return errors.New("validation failed")
	}
	return nil
}

func TestValidate(t *testing.T) {
	t.Run("all valid", func(t *testing.T) {
		v1 := &mockValidator[int]{valid: true}
		v2 := &mockValidator[int]{valid: true}
		v3 := &mockValidator[int]{valid: true}

		err := always.Validate[int](0, v1.Validate, v2.Validate, v3.Validate)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("multiple invalid", func(t *testing.T) {
		v1 := &mockValidator[int]{valid: true}
		v2 := &mockValidator[int]{valid: false}
		v3 := &mockValidator[int]{valid: true}

		err := always.Validate(0, v1.Validate, v2.Validate, v3.Validate)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("one invalid", func(t *testing.T) {
		v1 := &mockValidator[int]{valid: true}
		v2 := &mockValidator[int]{valid: false}
		v3 := &mockValidator[int]{valid: true}

		err := always.Validate(0, v1.Validate, v2.Validate, v3.Validate)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
