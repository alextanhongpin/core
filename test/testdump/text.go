package testdump

import "github.com/alextanhongpin/core/internal"

func Text(fileName string, str string, opt *TextOption) error {
	if opt == nil {
		opt = new(TextOption)
	}
	type T = string
	s := snapshot[T]{
		Marshaller:   MarshalFunc[string](MarshalText),
		Unmarshaller: UnmarshalFunc[string](UnmarshalText),
		Comparer:     CompareFunc[string](CompareText),
	}

	return Snapshot(fileName, str, &s, opt.Hooks...)
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
