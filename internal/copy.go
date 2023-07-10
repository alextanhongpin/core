package internal

import (
	"errors"

	"github.com/mitchellh/copystructure"
)

var ErrCopyFailed = errors.New("copy failed at type assertion")

func Copy[T any](t T) (T, error) {
	v, err := copystructure.Copy(t)
	if err != nil {
		return t, err
	}

	t, ok := v.(T)
	if !ok {
		return t, ErrCopyFailed
	}

	return t, nil
}
