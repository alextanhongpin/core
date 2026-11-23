package ahocorasick

// Match is one occurrence of a pattern: text[Start:End] equals
// patterns[PatternIndex].
type Match struct {
	Start        int
	End          int
	PatternIndex int
}

// Matcher represents the compiled Aho-Corasick automaton.
type Matcher struct {
	// Flat transition table: transitions[state * 256 + byte] -> nextState
	transitions []uint32
	// Fail links for BFS construction
	fail []uint32

	// outputs[state] lists the patterns ending at state, longest first.
	outputs [][]int32

	patternLens []int // length of each pattern by index
}

// NewMatcher compiles patterns into a Matcher. Empty patterns never match.
// The automaton uses 1 KiB per trie node, about one node per pattern byte.
// The patterns are not retained.
func NewMatcher(patterns [][]byte) *Matcher {
	m := &Matcher{
		patternLens: make([]int, len(patterns)),
	}

	// 1. Initial allocation for root state (State 0)
	m.addState()

	// 2. Build Trie
	for patIdx, pat := range patterns {
		m.patternLens[patIdx] = len(pat)
		if len(pat) == 0 {
			continue
		}

		var curr uint32 = 0
		for _, b := range pat {
			next := m.transitions[curr*256+uint32(b)]
			if next == 0 {
				next = m.addState()
				m.transitions[curr*256+uint32(b)] = next
			}
			curr = next
		}
		m.outputs[curr] = append(m.outputs[curr], int32(patIdx))
	}

	// 3. Build Failure Links & Compile Full DFA Transitions
	m.buildDFA()

	return m
}

func (m *Matcher) addState() uint32 {
	state := uint32(len(m.fail))
	m.fail = append(m.fail, 0)
	m.outputs = append(m.outputs, nil)

	// Allocate 256 byte transitions for the new state
	m.transitions = append(m.transitions, make([]uint32, 256)...)
	return state
}

func (m *Matcher) buildDFA() {
	queue := make([]uint32, 0, len(m.fail))

	// Queue depth-1 states
	for b := range 256 {
		child := m.transitions[b]
		if child != 0 {
			queue = append(queue, child)
		}
	}

	// BFS for failure link computation & transition compilation
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		failState := m.fail[curr]

		for b := range 256 {
			idx := curr*256 + uint32(b)
			child := m.transitions[idx]

			if child != 0 {
				// Set fail link of child to transition of failState
				m.fail[child] = m.transitions[failState*256+uint32(b)]

				// Patterns ending at the fail state are shorter suffixes.
				m.outputs[child] = append(m.outputs[child], m.outputs[m.fail[child]]...)

				queue = append(queue, child)
			} else {
				// Optimization: Flatten transition graph into direct DFA
				m.transitions[idx] = m.transitions[failState*256+uint32(b)]
			}
		}
	}
}

// FindAll returns every match in text, including overlapping ones, in the
// order MatchFunc reports them.
func (m *Matcher) FindAll(text []byte) []Match {
	var matches []Match
	m.MatchFunc(text, func(match Match) bool {
		matches = append(matches, match)
		return true // continue search
	})
	return matches
}

// MatchFunc calls fn for every match in text, ordered by End and, for
// matches with the same End, longest first, then by pattern index.
// Returning false from fn stops the scan. A Matcher is safe for concurrent
// use.
func (m *Matcher) MatchFunc(text []byte, fn func(Match) bool) {
	var state uint32 = 0

	for i, b := range text {
		state = m.transitions[state*256+uint32(b)]

		for _, patIdx := range m.outputs[state] {
			match := Match{
				Start:        i - m.patternLens[patIdx] + 1,
				End:          i + 1,
				PatternIndex: int(patIdx),
			}
			if !fn(match) {
				return
			}
		}
	}
}
