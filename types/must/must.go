package must

import (
	"fmt"
)

func Value[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Errorf("must: %w", err))
	}
	return v
}

func Nil(err error) {
	if err != nil {
		panic(fmt.Errorf("must: %w", err))
	}
}
