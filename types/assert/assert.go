package assert

func Assert(is bool, msg string) string {
	if is {
		return ""
	}

	return msg
}

func Required[T comparable](v T, assertions ...string) []string {
	var zero T
	if v != zero {
		return assertions
	}
	return NonZeroSlice(append([]string{"required"}, assertions...))
}

func Optional[T comparable](v T, assertions ...string) []string {
	var zero T
	if v == zero {
		return nil
	}

	return NonZeroSlice(assertions)
}

func NonZeroSlice[T comparable](vs []T) []T {
	var zero T
	res := make([]T, 0, len(vs))
	for _, v := range vs {
		if v == zero {
			continue
		}
		res = append(res, v)
	}

	return res
}

func NonZeroMap[K, V comparable](kv map[K]V) map[K]V {
	var zero V
	res := make(map[K]V)
	for k, v := range kv {
		if v == zero {
			continue
		}
		res[k] = v
	}

	return res
}
