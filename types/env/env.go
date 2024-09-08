package env

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var Error = fmt.Errorf("env")

type Parseable interface {
	~string | ~bool | ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~complex64 | ~complex128
}

func Parse[T Parseable](str string) (T, error) {
	var v T
	_, err := fmt.Sscanf(str, "%v", &v)
	if err != nil {
		return v, fmt.Errorf("%w: parse failed: %s", Error, err)
	}

	return v, nil
}

func LoadDuration(name string) time.Duration {
	s, err := lookupEnv(name)
	if err != nil {
		panic(err)
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}

	return d
}

func Load[T Parseable](name string) T {
	s, err := lookupEnv(name)
	if err != nil {
		panic(err)
	}

	v, err := Parse[T](strings.TrimSpace(s))
	if err != nil {
		panic(fmt.Errorf("%w: %s", err, name))
	}

	return v
}

func LoadSlice[T Parseable](name string, sep string) []T {
	v, err := lookupEnv(name)
	if err != nil {
		panic(err)
	}

	vs := strings.Split(v, sep)
	res := make([]T, len(vs))
	for i, s := range vs {
		v, err := Parse[T](strings.TrimSpace(s))
		if err != nil {
			panic(fmt.Errorf("%w: %s", err, name))
		}

		res[i] = v
	}

	return res
}

func lookupEnv(name string) (string, error) {
	v, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("%w: %q not set", Error, name)
	}

	return v, nil
}
