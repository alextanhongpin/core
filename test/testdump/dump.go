package testdump

import (
	"errors"
	"os"

	"github.com/alextanhongpin/core/internal"
)

type Marshaller[T any] interface {
	Marshal(T) ([]byte, error)
}

type Unmarshaller[T any] interface {
	Unmarshal([]byte) (T, error)
}

type Comparer[T any] interface {
	Compare(snapshot, received T) error
}

type snapshot[T any] struct {
	Marshaller[T]
	Unmarshaller[T]
	Comparer[T]
}

type S[T any] interface {
	Marshaller[T]
	Unmarshaller[T]
	Comparer[T]
}

type Hook[T any] func(S[T]) S[T]

func Snapshot[T any](fileName string, t T, s S[T], hooks ...Hook[T]) error {
	/*
		// Create a new copy.
		v, err := copystructure.Copy(t)
		if err != nil {
			return err
		}
		t = v.(T)
	*/

	// Run middleware in reverse order, so that the first
	// will execute first.
	for i := 0; i < len(hooks); i++ {
		mw := hooks[len(hooks)-i-1]
		s = mw(s)
	}

	b, err := s.Marshal(t)
	if err != nil {
		return err
	}

	if err := internal.WriteIfNotExists(fileName, b); err != nil {
		return err
	}

	// NOTE: We unmarshal back the bytes, since there might
	// be additional information not present during the
	// marshalling process.
	received, err := s.Unmarshal(b)
	if err != nil {
		return err
	}

	b, err = os.ReadFile(fileName)
	if err != nil {
		return err
	}

	snapshot, err := s.Unmarshal(b)
	if err != nil {
		return err
	}

	return s.Compare(snapshot, received)
}

type MarshalFunc[T any] (func(T) ([]byte, error))

func (f MarshalFunc[T]) Marshal(t T) ([]byte, error) {
	return f(t)
}

type MarshalFuncType[T, V any] (func(V) ([]byte, error))

func (f MarshalFuncType[T, V]) Marshal(t T) ([]byte, error) {
	v, ok := any(t).(V)
	if ok {
		return f(v)
	}

	return nil, errors.New("type casting failed")
}

type UnmarshalFunc[T any] (func([]byte) (T, error))

func (f UnmarshalFunc[T]) Unmarshal(b []byte) (T, error) {
	return f(b)
}

type CompareFunc[T any] (func(a, b T) error)

func (f CompareFunc[T]) Compare(a, b T) error {
	return f(a, b)
}

func MarshalHook[T any](hook func(T) (T, error)) Hook[T] {
	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S:    s,
			hook: hook,
		}
	}
}

func MarshalHookAny[T, V any](hook func(V) (V, error)) Hook[T] {
	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S: s,
			hook: func(t T) (T, error) {
				v, err := hook(any(t).(V))
				if err != nil {
					return t, err
				}

				return any(v).(T), nil
			},
		}
	}
}

func CompareHook[T any](hook func(snapshot T, received T) error) Hook[T] {
	return func(s S[T]) S[T] {
		return &compareHook[T]{
			S:    s,
			hook: hook,
		}
	}
}

type marshalHook[T any] struct {
	S[T]
	hook func(t T) (T, error)
}

func (m *marshalHook[T]) Marshal(t T) ([]byte, error) {
	if m.hook != nil {
		var err error
		t, err = m.hook(t)
		if err != nil {
			return nil, err
		}
	}
	return m.S.Marshal(t)
}

type compareHook[T any] struct {
	S[T]
	hook func(snapshot, received T) error
}

func (m *compareHook[T]) Compare(snapshot, received T) error {
	if m.hook != nil {
		if err := m.hook(snapshot, received); err != nil {
			return err
		}
	}

	return m.S.Compare(snapshot, received)
}
