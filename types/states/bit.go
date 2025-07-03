package states

// Predicate is a function that returns a boolean value.
type Predicate func() bool

// ExactlyN returns true if exactly n conditions are true.
// This is useful for checking polymorphic conditions where
// you need a specific number of conditions to be satisfied.
func ExactlyN(n int, conditions ...bool) bool {
	count := 0
	for _, condition := range conditions {
		if condition {
			count++
		}
	}
	return count == n
}

// ExactlyNFunc returns true if exactly n predicates return true.
func ExactlyNFunc(n int, predicates ...Predicate) bool {
	count := 0
	for _, predicate := range predicates {
		if predicate() {
			count++
		}
	}
	return count == n
}

// XOR returns true if exactly one condition is true.
// This is an alias for ExactlyN(1, conditions...).
func XOR(conditions ...bool) bool {
	return ExactlyN(1, conditions...)
}

// XORFunc returns true if exactly one predicate returns true.
func XORFunc(predicates ...Predicate) bool {
	return ExactlyNFunc(1, predicates...)
}

// AllOrNone returns true if all conditions are true or all are false.
// This is useful for checking if a set of fields are all set or none are set.
func AllOrNone(conditions ...bool) bool {
	count := 0
	for _, condition := range conditions {
		if condition {
			count++
		}
	}
	return count == 0 || count == len(conditions)
}

// AllOrNoneFunc returns true if all predicates return true or all return false.
func AllOrNoneFunc(predicates ...Predicate) bool {
	count := 0
	for _, predicate := range predicates {
		if predicate() {
			count++
		}
	}
	return count == 0 || count == len(predicates)
}

// AtLeastN returns true if at least n conditions are true.
func AtLeastN(n int, conditions ...bool) bool {
	count := 0
	for _, condition := range conditions {
		if condition {
			count++
		}
		if count >= n {
			return true
		}
	}
	return false
}

// AtLeastNFunc returns true if at least n predicates return true.
func AtLeastNFunc(n int, predicates ...Predicate) bool {
	count := 0
	for _, predicate := range predicates {
		if predicate() {
			count++
		}
		if count >= n {
			return true
		}
	}
	return false
}

// AtMostN returns true if at most n conditions are true.
func AtMostN(n int, conditions ...bool) bool {
	count := 0
	for _, condition := range conditions {
		if condition {
			count++
		}
		if count > n {
			return false
		}
	}
	return true
}

// AtMostNFunc returns true if at most n predicates return true.
func AtMostNFunc(n int, predicates ...Predicate) bool {
	count := 0
	for _, predicate := range predicates {
		if predicate() {
			count++
		}
		if count > n {
			return false
		}
	}
	return true
}

// Majority returns true if more than half of the conditions are true.
func Majority(conditions ...bool) bool {
	if len(conditions) == 0 {
		return false
	}
	required := (len(conditions) / 2) + 1
	return AtLeastN(required, conditions...)
}

// MajorityFunc returns true if more than half of the predicates return true.
func MajorityFunc(predicates ...Predicate) bool {
	if len(predicates) == 0 {
		return false
	}
	required := (len(predicates) / 2) + 1
	return AtLeastNFunc(required, predicates...)
}

// Any returns true if any condition is true.
func Any(conditions ...bool) bool {
	for _, condition := range conditions {
		if condition {
			return true
		}
	}
	return false
}

// AnyFunc returns true if any predicate returns true.
func AnyFunc(predicates ...Predicate) bool {
	for _, predicate := range predicates {
		if predicate() {
			return true
		}
	}
	return false
}

// All returns true if all conditions are true.
func All(conditions ...bool) bool {
	for _, condition := range conditions {
		if !condition {
			return false
		}
	}
	return len(conditions) > 0
}

// AllFunc returns true if all predicates return true.
func AllFunc(predicates ...Predicate) bool {
	for _, predicate := range predicates {
		if !predicate() {
			return false
		}
	}
	return len(predicates) > 0
}
