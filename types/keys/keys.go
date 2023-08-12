package keys

import "fmt"

type Key string

func New(k string) Key {
	return Key(k)
}

func (k Key) Format(args ...any) string {
	return fmt.Sprintf(string(k), args...)
}
