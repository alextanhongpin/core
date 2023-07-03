package sliceutil

import "golang.org/x/exp/constraints"

func Map[K, V any](ks []K, fn func(i int) V) []V {
	vs := make([]V, len(ks))
	for i := 0; i < len(ks); i++ {
		vs[i] = fn(i)
	}

	return vs
}

func MapError[K, V any](ks []K, fn func(i int) (V, error)) ([]V, error) {
	vs := make([]V, len(ks))
	for i := 0; i < len(ks); i++ {
		v, err := fn(i)
		if err != nil {
			return nil, err
		}

		vs[i] = v
	}

	return vs, nil
}

func Filter[V any](vs []V, fn func(i int) bool) []V {
	res := make([]V, 0, len(vs))
	for i := 0; i < len(vs); i++ {
		if !fn(i) {
			continue
		}

		res = append(res, vs[i])
	}

	return res
}

func Sum[T constraints.Integer](n []T) (total T) {
	for i := 0; i < len(n); i++ {
		total += n[i]
	}

	return total
}

func Min[T constraints.Integer](n []T) T {
	if len(n) == 0 {
		return 0
	}

	min := n[0]
	for i := 1; i < len(n); i++ {
		if n[i] < min {
			min = n[i]
		}
	}

	return min
}

func Max[T constraints.Integer](n []T) T {
	if len(n) == 0 {
		return 0
	}

	max := n[0]
	for i := 1; i < len(n); i++ {
		if n[i] > max {
			max = n[i]
		}
	}

	return max
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
