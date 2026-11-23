package states

// count returns how many conditions are true.
func count(conditions []bool) int {
	n := 0
	for _, c := range conditions {
		if c {
			n++
		}
	}
	return n
}

// ExactlyN reports whether exactly n conditions are true.
func ExactlyN(n int, conditions ...bool) bool {
	return count(conditions) == n
}

// ExactlyOne reports whether exactly one condition is true, for fields that
// are mutually exclusive but required (for example a party that must be
// either a person or an organization). Unlike a chained ^, it is false when
// three of three conditions are true.
func ExactlyOne(conditions ...bool) bool {
	return ExactlyN(1, conditions...)
}

// AtLeastN reports whether n or more conditions are true. It is true for
// n <= 0.
func AtLeastN(n int, conditions ...bool) bool {
	return count(conditions) >= n
}

// AtMostN reports whether no more than n conditions are true. It is false
// for n < 0.
func AtMostN(n int, conditions ...bool) bool {
	return count(conditions) <= n
}

// AllOrNone reports whether the conditions are all true or all false, for
// fields that must be set together. It is true for no conditions.
func AllOrNone(conditions ...bool) bool {
	n := count(conditions)
	return n == 0 || n == len(conditions)
}

// Majority reports whether more than half of the conditions are true. It is
// false for no conditions and for a tie.
func Majority(conditions ...bool) bool {
	return count(conditions)*2 > len(conditions)
}
