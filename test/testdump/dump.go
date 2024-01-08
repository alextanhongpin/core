package testdump

import (
	"os"

	"github.com/alextanhongpin/core/internal"
)

type Hook[T any] func(S[T]) S[T]

type Hooks[T any] []Hook[T]

func (hooks Hooks[T]) Apply(s S[T]) S[T] {
	// Reverse the hooks so that it applies from left to right.
	for i := 0; i < len(hooks); i++ {
		h := hooks[len(hooks)-i-1]
		s = h(s)
	}

	return s
}

func Snapshot[T any](rw readerWriter, t T, s S[T]) error {
	b, err := s.Marshal(t)
	if err != nil {
		return err
	}

	if err := rw.Write(b); err != nil {
		return err
	}

	// NOTE: We unmarshal back the bytes, since there might
	// be additional information not present during the
	// marshalling process.
	received, err := s.Unmarshal(b)
	if err != nil {
		return err
	}

	b, err = rw.Read()
	if err != nil {
		return err
	}

	snapshot, err := s.Unmarshal(b)
	if err != nil {
		return err
	}

	return s.Compare(snapshot, received)
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

type File struct {
	Name string
}

func NewFile(name string) *File {
	return &File{
		Name: name,
	}
}

func (rw *File) Read() ([]byte, error) {
	return os.ReadFile(rw.Name)
}

func (rw *File) Write(b []byte) error {
	return internal.WriteFile(rw.Name, b, *update)
}

type InMemory struct {
	Idempotent bool
	Data       []byte
}

func NewInMemory() *InMemory {
	return &InMemory{
		Idempotent: true,
		Data:       nil,
	}
}

func (rw *InMemory) Read() ([]byte, error) {
	return rw.Data, nil
}

func (rw *InMemory) Write(b []byte) error {
	if rw.Idempotent && len(rw.Data) > 0 {
		return nil
	}

	rw.Data = b

	return nil
}
