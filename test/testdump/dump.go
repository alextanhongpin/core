package testdump

import (
	"bytes"
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

type S[T any] interface {
	Marshaller[T]
	Unmarshaller[T]
	Comparer[T]
}

type Hook[T any] func(S[T]) S[T]

func Snapshot[T any](fileName string, t T, ss *snapshot[T], hooks ...Hook[T]) error {
	var s S[T] = ss
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

	receivedBytes := bytes.Clone(b)
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

	snapshotBytes := bytes.Clone(b)
	snapshot, err := s.Unmarshal(b)
	if err != nil {
		return err
	}

	// This is required when comparing JSON/YAML type, because
	// unmarshalling the type to map[any]interface{} will cause
	// information to be lost (e.g. additional fields).
	if ss.unmarshalAny != nil && ss.compareAny != nil {
		x, err := ss.unmarshalAny.Unmarshal(snapshotBytes)
		if err != nil {
			return err
		}

		y, err := ss.unmarshalAny.Unmarshal(receivedBytes)
		if err != nil {
			return err
		}

		if err := ss.compareAny.Compare(x, y); err != nil {
			return err
		}
	}

	return s.Compare(snapshot, received)
}

type MarshalFunc[T any] (func(T) ([]byte, error))

func (f MarshalFunc[T]) Marshal(t T) ([]byte, error) {
	return f(t)
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

func CompareHook[T any](hook func(snapshot T, received T) error) Hook[T] {
	return func(s S[T]) S[T] {
		return &compareHook[T]{
			S:    s,
			hook: hook,
		}
	}
}

type snapshot[T any] struct {
	Marshaller[T]
	Unmarshaller[T]
	Comparer[T]

	unmarshalAny Unmarshaller[any]
	compareAny   Comparer[any]
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

func nopComparer[T any](a, b T) error {
	return nil
}

func Copier[T any]() Hook[T] {
	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S: s,
			hook: func(t T) (T, error) {
				return internal.Copy(t)
			},
		}
	}
}
