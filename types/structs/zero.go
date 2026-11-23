package structs

import "reflect"

// IsNilOrZero reports whether x is nil, a typed nil (pointer, map, slice,
// channel, func), or the zero value of its type. A non-nil empty slice or map
// is not zero.
func IsNilOrZero(x any) bool {
	return x == nil || reflect.ValueOf(x).IsZero()
}
