package ahocorasick

// Match represents a pattern match found in the input text.
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

	// Compressed output list representation
	outputHead []int32 // state -> index into outputPatterns (-1 if no match)
	outputList []int32 // pattern indices stored in linked list form
	outputNext []int32 // pointer to next pattern index for the state (-1 if end)

	patternLens []int // length of each pattern by index
}

// NewMatcher constructs and compiles an Aho-Corasick automaton from a slice of patterns.
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
		m.addOutput(curr, int32(patIdx))
	}

	// 3. Build Failure Links & Compile Full DFA Transitions
	m.buildDFA()

	return m
}

func (m *Matcher) addState() uint32 {
	state := uint32(len(m.fail))
	m.fail = append(m.fail, 0)
	m.outputHead = append(m.outputHead, -1)

	// Allocate 256 byte transitions for the new state
	m.transitions = append(m.transitions, make([]uint32, 256)...)
	return state
}

func (m *Matcher) addOutput(state uint32, patternIdx int32) {
	head := m.outputHead[state]
	newHead := int32(len(m.outputList))

	m.outputList = append(m.outputList, patternIdx)
	m.outputNext = append(m.outputNext, head)
	m.outputHead[state] = newHead
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

				// Merge outputs from fail state into child state
				m.mergeOutputs(child, m.fail[child])

				queue = append(queue, child)
			} else {
				// Optimization: Flatten transition graph into direct DFA
				m.transitions[idx] = m.transitions[failState*256+uint32(b)]
			}
		}
	}
}

func (m *Matcher) mergeOutputs(toState, fromState uint32) {
	outIdx := m.outputHead[fromState]
	for outIdx != -1 {
		patIdx := m.outputList[outIdx]
		m.addOutput(toState, patIdx)
		outIdx = m.outputNext[outIdx]
	}
}

// FindAll scans text and returns all matches.
func (m *Matcher) FindAll(text []byte) []Match {
	var matches []Match
	m.MatchFunc(text, func(match Match) bool {
		matches = append(matches, match)
		return true // continue search
	})
	return matches
}

// MatchFunc streams matches through a callback to eliminate slice allocations.
// Returning false from the callback stops scanning immediately.
func (m *Matcher) MatchFunc(text []byte, fn func(Match) bool) {
	var state uint32 = 0

	for i, b := range text {
		state = m.transitions[state*256+uint32(b)]

		outIdx := m.outputHead[state]
		for outIdx != -1 {
			patIdx := m.outputList[outIdx]
			patLen := m.patternLens[patIdx]

			match := Match{
				Start:        i - patLen + 1,
				End:          i + 1,
				PatternIndex: int(patIdx),
			}

			if !fn(match) {
				return
			}

			outIdx = m.outputNext[outIdx]
		}
	}
}
