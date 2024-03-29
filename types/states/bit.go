package states

// XOR returns true if at least n conditional is true.
// Useful for checking polymorphic conditions.
func XOR(n int, ts ...bool) bool {
	var success int
	for _, t := range ts {
		if t {
			success++
		}
	}

	return success == n
}

func XORFunc(n int, hs ...Handler) bool {
	var success int
	for _, h := range hs {
		if h() {
			success++
		}
	}

	return success == n
}

// AllOrNone returns true if all condition is true, or all
// is false.
// Useful to check if a set of fields are all set or none
// are set.
func AllOrNone(ts ...bool) bool {
	var success int
	for _, t := range ts {
		if t {
			success++
		}
	}

	return success == 0 || success == len(ts)
}

func AllOrNoneFunc(hs ...Handler) bool {
	var success int
	for _, h := range hs {
		if h() {
			success++
		}
	}

	return success == 0 || success == len(hs)
}
