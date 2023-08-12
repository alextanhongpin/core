package generics

import "encoding/json"

func Ptr[T any](v T) *T {
	return &v
}

func As[T any](unk any) (T, bool) {
	v, ok := unk.(T)
	return v, ok
}

type JSON[T any] string

func (j JSON[T]) Marshal(v T) ([]byte, error) {
	return json.Marshal(v)
}

func (j JSON[T]) Unmarshal(b []byte) (T, error) {
	var v T
	err := json.Unmarshal(b, &v)
	return v, err
}
