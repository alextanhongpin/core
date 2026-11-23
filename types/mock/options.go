package mock

// Map holds values per method name.
type Map[T any] map[string][]T

// Values returns the values stored for key.
func (m Map[T]) Values(key string) []T {
	return m[key]
}

// With returns a copy of m with vals appended to key. m is not modified.
func (m Map[T]) With(key string, vals ...T) Map[T] {
	cloned := m.clone()
	cloned[key] = append(cloned[key], vals...)
	return cloned
}

// Del removes key from m.
func (m Map[T]) Del(key string) {
	delete(m, key)
}

func (m Map[T]) clone() Map[T] {
	res := make(Map[T], len(m))
	for k, v := range m {
		res[k] = append([]T(nil), v...)
	}
	return res
}

// Options maps a method name to the strings Call returns, in order.
type Options = Map[string]

// Calls maps a method name to the arguments of each call.
type Calls = Map[[]any]
