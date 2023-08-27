package testdump

import "github.com/alextanhongpin/core/internal"

type TextOption struct{}

func Text(rw readerWriter, str string, opt *TextOption, hooks ...Hook[string]) error {
	type T = string
	var s S[T] = &snapshot[T]{
		marshaler:   MarshalFunc[string](MarshalText),
		unmarshaler: UnmarshalFunc[string](UnmarshalText),
		comparer:    CompareFunc[string](CompareText),
	}

	s = Hooks[T](hooks).Apply(s)

	return Snapshot(rw, str, s)
}

func MarshalText(str string) ([]byte, error) {
	return []byte(str), nil
}

func UnmarshalText(b []byte) (string, error) {
	return string(b), nil
}

func CompareText(snapshot, received string) error {
	return internal.ANSIDiff(snapshot, received)
}
