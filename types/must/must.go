package must

import "log"

func Value[T any](v T, err error) T {
	if err != nil {
		log.Fatalf("must: %v", err)
	}
	return v
}

func Nil(err error) {
	if err != nil {
		log.Fatalf("must: %v", err)
	}
}
