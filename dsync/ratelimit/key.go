package ratelimit

import "fmt"

type Key string

func (k Key) Format(args ...any) string {
	return fmt.Sprintf(string(k), args...)
}
