// package sliceutil covers utilities not found in
// https://pkg.go.dev/golang.org/x/exp/slices
package sliceutil

func Map[K, V any](ks []K, fn func(i int) V) []V {
	vs := make([]V, len(ks))
	for i := range len(ks) {
		vs[i] = fn(i)
	}

	return vs
}

func MapError[K, V any](ks []K, fn func(i int) (V, error)) ([]V, error) {
	vs := make([]V, len(ks))
	for i := range len(ks) {
		v, err := fn(i)
		if err != nil {
			return nil, err
		}

		vs[i] = v
	}

	return vs, nil
}

func Dedup[T comparable](t []T) []T {
	cache := make(map[T]bool)
	for i := range t {
		cache[t[i]] = true
	}

	unique := make([]T, 0, len(cache))
	for key := range cache {
		unique = append(unique, key)
	}

	return unique
}

func DedupFunc[T any, K comparable](t []T, fn func(i int) K) []T {
	res := make([]T, 0)

	seen := make(map[K]bool)
	for i := range len(t) {
		k := fn(i)
		if seen[k] {
			continue
		}
		seen[k] = true
		res = append(res, t[i])
	}

	return res
}
