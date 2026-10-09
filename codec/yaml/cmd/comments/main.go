package main

import (
	"fmt"
	"log"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Development map[string]any `yaml:"development"`
	Production  map[string]any `yaml:"production"`
}

func main() {
	yamlInput := `
development:
  # &db_properties
  host: "localhost"
  port: 5432
  user: "dev_user"

# &items
items:
	- 1
	- 2
	- 3

production:
  # *db_properties
  user: "prod_admin"
	# *items
`

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yamlInput), &root); err != nil {
		log.Fatalf("Unmarshal error: %v", err)
	}

	// Registry to map anchor strings -> Deep copied YAML nodes
	anchorRegistry := make(map[string]*yaml.Node)

	// Step 1: Find and register anchors
	findCommentAnchors(&root, anchorRegistry)

	// Step 2: Replace comment aliases with registered nodes
	resolveCommentAliases(&root, anchorRegistry)

	// Step 3: Decode into final Go types
	var cfg Config
	if err := root.Decode(&cfg); err != nil {
		log.Fatalf("Decode error: %v", err)
	}

	fmt.Printf("Resolved Config Structure:\n%+v\n", cfg)
}

// Recursively checks nodes for "# &anchor_name"
func findCommentAnchors(node *yaml.Node, registry map[string]*yaml.Node) {
	// HeadComment holds any comments in the lines preceding the node
	if node.HeadComment != "" {
		comment := strings.TrimSpace(node.HeadComment)
		if strings.HasPrefix(comment, "# &") {
			anchorName := strings.TrimPrefix(comment, "# &")
			// Store a clone of this node mapped to the anchor string
			registry[anchorName] = cloneNode(node)
		}
	}

	for _, child := range node.Content {
		findCommentAnchors(child, registry)
	}
}

// Recursively scans and injects the contents where "# *anchor_name" is encountered
func resolveCommentAliases(node *yaml.Node, registry map[string]*yaml.Node) {
	if node.Kind == yaml.MappingNode {
		var newContent []*yaml.Node

		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valNode := node.Content[i+1]

			// Check if the key contains an alias comment indicating an inherit/merge
			if keyNode.HeadComment != "" {
				comment := strings.TrimSpace(keyNode.HeadComment)
				if strings.HasPrefix(comment, "# *") {
					aliasName := strings.TrimPrefix(comment, "# *")

					if sourceNode, exists := registry[aliasName]; exists && sourceNode.Kind == yaml.MappingNode {
						// Append all key-values from our virtual comment anchor
						newContent = append(newContent, sourceNode.Content...)
					}
				}
			}

			newContent = append(newContent, keyNode, valNode)
		}
		node.Content = deduplicateKeys(newContent)
	}

	for _, child := range node.Content {
		resolveCommentAliases(child, registry)
	}
}

// Helper to deep-copy a Node so modifications don't break the original tree reference
func cloneNode(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	cp := *n
	if len(n.Content) > 0 {
		cp.Content = make([]*yaml.Node, len(n.Content))
		for i, child := range n.Content {
			cp.Content[i] = cloneNode(child)
		}
	}
	return &cp
}

// Deduplicates map elements to follow standard last-one-wins properties
func deduplicateKeys(content []*yaml.Node) []*yaml.Node {
	var finalContent []*yaml.Node
	tracker := make(map[string]int)

	for i := 0; i < len(content); i += 2 {
		k := content[i]
		v := content[i+1]

		if idx, found := tracker[k.Value]; found {
			finalContent[idx+1] = v
		} else {
			tracker[k.Value] = len(finalContent)
			finalContent = append(finalContent, k, v)
		}
	}
	return finalContent
}
