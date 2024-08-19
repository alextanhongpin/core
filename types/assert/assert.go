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
	return append([]string{"required"}, assertions...)
}

func Optional[T comparable](v T, assertions ...string) []string {
	var zero T
	if v == zero {
		return nil
	}

	return assertions
}
