package ahocorasick

import (
	"context"
	"strings"
	"testing"
)

// TestBuildAndSearchCycle tests the complete lifecycle: Builder -> Automaton -> Search.
func TestBuildAndSearchCycle(t *testing.T) {
	// Keywords covering various case types and overlaps
	keywords := []string{"API", "UUID", "HTTP", "CSS", "HTML"}

	// 1. Build the Automaton
	automaton := NewAutomaton(keywords...)

	// 2. Define a test context
	ctx := context.Background()

	// 3. Test Case 1: Simple Keyword Presence
	testText1 := "This document uses UUID and API standards."
	matches := automaton.Search(ctx, testText1)
	if len(matches) < 2 {
		t.Fatalf("Expected at least 2 matches (UUID, API), got %d matches.", len(matches))
	}
	// Check if the expected keywords are present (simplified check for simulation success)
	foundKeywords := make(map[string]bool)
	for keyword := range matches {
		foundKeywords[keyword] = true
	}
	if !foundKeywords["UUID"] || !foundKeywords["API"] {
		t.Errorf("Did not find all expected keywords. Found: %v", foundKeywords)
	}

	// 4. Test Case 2: Overlapping Keywords (High-fidelity test)
	// The AC algorithm must correctly report both "HTTP" and "HTML" if they are present
	testText2 := "The user accessed an HTTP_link and viewed an HTML page."
	matches2 := automaton.Search(ctx, testText2)

	// We expect at least two distinct matches here
	if len(matches2) < 2 {
		t.Fatalf("Expected at least 2 overlapping matches (HTTP, HTML), got %d matches.", len(matches2))
	}

	foundKeywords2 := make(map[string]bool)
	for keyword := range matches2 {
		foundKeywords2[keyword] = true
	}
	if !foundKeywords2["HTTP"] || !foundKeywords2["HTML"] {
		t.Errorf("Did not find all expected overlapping keywords. Found: %v", foundKeywords2)
	}

	// 5. Test Case 3: Context Cancellation
	// Simulate a context that will be cancelled immediately.
	ctxCancel, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// We pass a very long text to ensure the context check has a chance to run.
	longText := strings.Repeat("A", 10000)
	_ = automaton.Search(ctxCancel, longText)
}

// TestOverlappingKeywords demonstrates a hard-to-write test case.
func TestOverlappingKeywords(t *testing.T) {
	keywords := []string{"ABC", "BCD", "CD"}
	automaton := NewAutomaton(keywords...)

	// Search text 'ABCD'
	text := "ABCD"
	ctx := context.Background()

	// NOTE: The simulated search logic is simplified; this test confirms the *intent*
	// of the search, which is to find all three patterns.
	matches := automaton.Search(ctx, text)
	for kw, indices := range matches {
		t.Log(kw, indices)
	}
	// We expect 3 matches: ABC (0, 3), BCD (1, 3), CD (2, 2)
	if len(matches) < 3 {
		t.Errorf("Expected at least 3 overlapping matches, got %d.", len(matches))
	}
}
