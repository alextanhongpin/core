package testdump

import "github.com/alextanhongpin/core/internal"

func Text(rw readerWriter, str string, opt *TextOption) error {
	if opt == nil {
		opt = new(TextOption)
	}

	type T = string
	var s S[T] = &snapshot[T]{
		marshaler:   MarshalFunc[string](MarshalText),
		unmarshaler: UnmarshalFunc[string](UnmarshalText),
		comparer:    CompareFunc[string](CompareText),
	}

	return Snapshot(rw, str, s, opt.Hooks...)
}

type TextOption struct {
	Hooks []Hook[string]
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
