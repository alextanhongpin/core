package testdump

type reader interface {
	Read() ([]byte, error)
}

type writer interface {
	Write([]byte) error
}

type readerWriter interface {
	reader
	writer
}

type marshaler[T any] interface {
	Marshal(T) ([]byte, error)
}

type unmarshaler[T any] interface {
	Unmarshal([]byte) (T, error)
}

type comparer[T any] interface {
	Compare(snapshot, received T) error
}

type anyUnmarshaler interface {
	UnmarshalAny([]byte) (any, error)
}

type anyComparer interface {
	CompareAny(snapshot, received any) error
}

type S[T any] interface {
	marshaler[T]
	unmarshaler[T]
	comparer[T]
}

type snapshot[T any] struct {
	marshaler[T]
	unmarshaler[T]
	comparer[T]
}

var _ S[any] = (*snapshot[any])(nil)

func (s *snapshot[T]) Compare(a, b T) error {
	if s.comparer == nil {
		return nil
	}

	return s.comparer.Compare(a, b)
}

type MarshalFunc[T any] (func(T) ([]byte, error))

func (f MarshalFunc[T]) Marshal(t T) ([]byte, error) {
	return f(t)
}

type UnmarshalFunc[T any] (func([]byte) (T, error))

func (f UnmarshalFunc[T]) Unmarshal(b []byte) (T, error) {
	return f(b)
}

type UnmarshalAnyFunc (func([]byte) (any, error))

func (f UnmarshalAnyFunc) UnmarshalAny(b []byte) (any, error) {
	return f(b)
}

type CompareFunc[T any] (func(a, b T) error)

func (f CompareFunc[T]) Compare(a, b T) error {
	return f(a, b)
}

type CompareAnyFunc (func(a, b any) error)

func (f CompareAnyFunc) CompareAny(a, b any) error {
	return f(a, b)
}
