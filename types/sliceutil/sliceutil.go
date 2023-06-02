package sliceutil

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
