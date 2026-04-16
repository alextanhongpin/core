package ahocorasick

import "context"

type TrieNode struct {
	output   []string
	children map[rune]*TrieNode
	fail     *TrieNode
}

func (n *TrieNode) hasChild(char rune) bool {
	_, ok := n.children[char]
	return ok
}
func NewTrieNode() *TrieNode {
	return &TrieNode{
		children: make(map[rune]*TrieNode),
	}
}

type Automaton struct {
	root     *TrieNode
	keywords []string
}

func NewAutomaton(keywords ...string) *Automaton {
	return &Automaton{
		root: buildAutomaton(keywords...),
	}
}

func buildAutomaton(keywords ...string) *TrieNode {
	// Initialize root node of the trie
	root := NewTrieNode()

	// Build trie
	for _, keyword := range keywords {
		node := root
		// Traverse the trie and create nodes for each character
		for _, char := range keyword {
			if _, ok := node.children[char]; !ok {
				node.children[char] = NewTrieNode()
			}
			node = node.children[char]
		}

		// Add keyword to the output list of the final node
		node.output = append(node.output, keyword)
	}

	// Build failure links using BFS
	var queue []*TrieNode

	// Start from root's children
	for _, node := range root.children {
		queue = append(queue, node)
		node.fail = root
	}

	// Breadth-first traversal of the trie
	for len(queue) > 0 {
		current_node := queue[0]
		queue = queue[1:]
		// Traverse each child node
		for key, next_node := range current_node.children {
			queue = append(queue, next_node)
			fail_node := current_node.fail
			// Find the longest proper suffix that is also a prefix
			for fail_node != nil && !fail_node.hasChild(key) {
				fail_node = fail_node.fail
			}
			// Set failure link of the current node
			if fail_node != nil {
				next_node.fail = fail_node.children[key]
			} else {
				next_node.fail = root
			}
			// Add output patterns of failure node to current node's output
			next_node.output = append(next_node.output, next_node.fail.output...)

		}
	}
	return root
}

func (a *Automaton) Search(ctx context.Context, text string) map[string][]int {
	root := a.root
	// Initialize result dictionary
	result := make(map[string][]int)

	current_node := root
	// Traverse the text
	for i, char := range text {
		// Follow failure links until a match is found
		for current_node != nil && !current_node.hasChild(char) {
			current_node = current_node.fail
		}

		if current_node == nil {
			current_node = root
			continue
		}

		// Move to the next node based on current character
		current_node = current_node.children[char]
		// Record matches found at this position
		for _, keyword := range current_node.output {
			result[keyword] = append(result[keyword], i-len(keyword)+1)
		}
	}
	return result
}
