package ahocorasick

import (
	"reflect"
	"sort"
	"testing"
)

// Helper function to turn strings into byte slices
func toBytes(strs []string) [][]byte {
	b := make([][]byte, len(strs))
	for i, s := range strs {
		b[i] = []byte(s)
	}
	return b
}

func TestStandardMatching(t *testing.T) {
	patterns := toBytes([]string{"he", "she", "his", "hers"})
	text := []byte("ushers")

	matcher := NewMatcher(patterns)
	matches := matcher.FindAll(text)

	expected := []Match{
		{Start: 2, End: 4, PatternIndex: 0}, // "he"
		{Start: 1, End: 4, PatternIndex: 1}, // "she"
		{Start: 2, End: 6, PatternIndex: 3}, // "hers"
	}

	if !reflect.DeepEqual(matches, expected) {
		t.Errorf("got %v, want %v", matches, expected)
	}
}

func TestOverlappingAndSubstrings(t *testing.T) {
	patterns := toBytes([]string{"a", "aa", "aaa", "aaaa"})
	text := []byte("aaaa")

	matcher := NewMatcher(patterns)
	matches := matcher.FindAll(text)

	// Sort matches by start position, then pattern index for deterministic check
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Start == matches[j].Start {
			return matches[i].PatternIndex < matches[j].PatternIndex
		}
		return matches[i].Start < matches[j].Start
	})

	expected := []Match{
		{Start: 0, End: 1, PatternIndex: 0}, // "a" at 0
		{Start: 0, End: 2, PatternIndex: 1}, // "aa" at 0..2
		{Start: 0, End: 3, PatternIndex: 2}, // "aaa" at 0..3
		{Start: 0, End: 4, PatternIndex: 3}, // "aaaa" at 0..4
		{Start: 1, End: 2, PatternIndex: 0}, // "a" at 1
		{Start: 1, End: 3, PatternIndex: 1}, // "aa" at 1..3
		{Start: 1, End: 4, PatternIndex: 2}, // "aaa" at 1..4
		{Start: 2, End: 3, PatternIndex: 0}, // "a" at 2
		{Start: 2, End: 4, PatternIndex: 1}, // "aa" at 2..4
		{Start: 3, End: 4, PatternIndex: 0}, // "a" at 3
	}

	if len(matches) != len(expected) {
		t.Fatalf("got %d matches, want %d", len(matches), len(expected))
	}

	for i := range expected {
		if matches[i] != expected[i] {
			t.Errorf("match[%d]: got %v, want %v", i, matches[i], expected[i])
		}
	}
}

func TestEarlyTermination(t *testing.T) {
	patterns := toBytes([]string{"foo", "bar", "baz"})
	text := []byte("foo bar baz")

	matcher := NewMatcher(patterns)

	var count int
	matcher.MatchFunc(text, func(m Match) bool {
		count++
		return false // Stop after the first match
	})

	if count != 1 {
		t.Errorf("expected 1 match before termination, got %d", count)
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("No Matches", func(t *testing.T) {
		patterns := toBytes([]string{"apple", "banana"})
		text := []byte("orange lemon cherry")

		matcher := NewMatcher(patterns)
		matches := matcher.FindAll(text)

		if len(matches) != 0 {
			t.Errorf("expected 0 matches, got %d", len(matches))
		}
	})

	t.Run("Empty Input Text", func(t *testing.T) {
		patterns := toBytes([]string{"foo", "bar"})
		matcher := NewMatcher(patterns)

		matches := matcher.FindAll([]byte(""))
		if len(matches) != 0 {
			t.Errorf("expected 0 matches, got %d", len(matches))
		}
	})

	t.Run("Empty Pattern Included", func(t *testing.T) {
		patterns := toBytes([]string{"foo", "", "bar"})
		matcher := NewMatcher(patterns)

		matches := matcher.FindAll([]byte("foobar"))
		expected := []Match{
			{Start: 0, End: 3, PatternIndex: 0},
			{Start: 3, End: 6, PatternIndex: 2},
		}

		if !reflect.DeepEqual(matches, expected) {
			t.Errorf("got %v, want %v", matches, expected)
		}
	})
}

// -----------------------------------------------------------------------------
// BENCHMARKS
// -----------------------------------------------------------------------------

func BenchmarkMatcherFindAll(b *testing.B) {
	patterns := toBytes([]string{
		"golang", "performance", "aho", "corasick", "algorithm",
		"byte", "slice", "memory", "cache", "flat",
	})
	text := []byte("a performant ahocorasick implementation in golang using flat memory structures and byte slices for optimal cache locality")

	matcher := NewMatcher(patterns)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = matcher.FindAll(text)
	}
}

func BenchmarkMatcherMatchFunc(b *testing.B) {
	patterns := toBytes([]string{
		"golang", "performance", "aho", "corasick", "algorithm",
		"byte", "slice", "memory", "cache", "flat",
	})
	text := []byte("a performant ahocorasick implementation in golang using flat memory structures and byte slices for optimal cache locality")

	matcher := NewMatcher(patterns)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		matcher.MatchFunc(text, func(m Match) bool {
			return true
		})
	}
}
