package envvar

import (
	"fmt"
	"os"
)

type Parseable interface {
	~string | ~bool | ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~complex64 | ~complex128
}

func Parse[T Parseable](str string) (T, error) {
	var result T
	_, err := fmt.Sscanf(str, "%v", &result)
	return result, err
}

func Load[T Parseable](name string) T {
	s, ok := os.LookupEnv(name)
	if ok == false {
		panic(fmt.Sprintf("envvar: %q not set", name))
	}
	v, err := Parse[T](s)
	if err != nil {
		panic(fmt.Sprintf("envvar: parse %q failed: %s", name, err))
	}
	return v
}
