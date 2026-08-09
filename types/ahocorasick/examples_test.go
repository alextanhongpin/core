package ahocorasick

import (
	"fmt"
)

func Example() {
	patterns := [][]byte{
		[]byte("he"),
		[]byte("she"),
		[]byte("his"),
		[]byte("hers"),
	}

	matcher := NewMatcher(patterns)
	text := []byte("ushers")

	// 1. Return as slice
	matches := matcher.FindAll(text)
	for _, match := range matches {
		fmt.Printf("Matched pattern #%d (%s) at [%d:%d]\n",
			match.PatternIndex,
			patterns[match.PatternIndex],
			match.Start,
			match.End,
		)
	}

	// 2. High-speed zero-allocation callback stream
	matcher.MatchFunc(text, func(m Match) bool {
		// Stop searching as soon as we find "she"
		if m.PatternIndex == 1 {
			fmt.Println("Found 'she', stopping early.")
			return false
		}
		return true
	})
	// Output:
	// Matched pattern #0 (he) at [2:4]
	// Matched pattern #1 (she) at [1:4]
	// Matched pattern #3 (hers) at [2:6]
	// Found 'she', stopping early.
}
