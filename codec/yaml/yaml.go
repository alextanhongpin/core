// Package yaml loads YAML with relative !include and ordered "!merge" directives.
package yaml

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	yamlv3 "gopkg.in/yaml.v3"
)

// LoadExtendedYAML resolves directives before decoding into out. Included paths
// are relative to the file containing the directive. Merges are shallow and
// processed in source order, with later values replacing earlier values.
// Only trusted files should be loaded: directives may read arbitrary paths.
func LoadExtendedYAML(filename string, out any) error {
	node, err := load(filename, make(map[string]bool), 0)
	if err != nil {
		return err
	}
	if err := node.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", filename, err)
	}
	return nil
}

func load(filename string, active map[string]bool, depth int) (*yamlv3.Node, error) {
	if depth >= 100 {
		return nil, fmt.Errorf("include depth exceeds 100 at %s", filename)
	}
	path, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	// Canonicalize symlinks so cycles through alternate names are detected.
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", filename, err)
	}
	if active[path] {
		return nil, fmt.Errorf("include cycle at %s", path)
	}
	active[path] = true
	defer delete(active, path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	decoder := yamlv3.NewDecoder(strings.NewReader(string(data)))
	var doc yamlv3.Node
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var extra yamlv3.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("%s: multiple YAML documents are unsupported", path)
		}
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("%s: empty YAML document", path)
	}
	if err := resolve(doc.Content[0], filepath.Dir(path), active, depth); err != nil {
		return nil, fmt.Errorf("resolve %s: %w", path, err)
	}
	return doc.Content[0], nil
}

func resolve(node *yamlv3.Node, dir string, active map[string]bool, depth int) error {
	if node.Tag == "!include" {
		names, err := filenames(node)
		if err != nil {
			return err
		}
		var merged *yamlv3.Node
		for _, name := range names {
			target, err := load(filepath.Join(dir, name), active, depth+1)
			if err != nil {
				return err
			}
			if merged == nil {
				merged = target
			} else if err := combine(merged, target); err != nil {
				return err
			}
		}
		*node = *merged // Preserve pointers used by native YAML aliases.
		return nil
	}
	if node.Kind == yamlv3.MappingNode {
		var content []*yamlv3.Node
		indices := make(map[string]int)
		seen := make(map[string]bool)
		appendPair := func(key, value *yamlv3.Node) {
			id := key.Tag + "\x00" + key.Value
			if index, ok := indices[id]; ok {
				content[index+1] = value
			} else {
				indices[id] = len(content)
				content = append(content, key, value)
			}
		}
		for i := 0; i < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			if key.Value == "!merge" || key.Tag == "!merge" {
				names, err := filenames(value)
				if err != nil {
					return err
				}
				for _, name := range names {
					target, err := load(filepath.Join(dir, name), active, depth+1)
					if err != nil {
						return err
					}
					if target.Kind != yamlv3.MappingNode {
						return fmt.Errorf("!merge %s must contain a mapping", name)
					}
					for j := 0; j < len(target.Content); j += 2 {
						appendPair(target.Content[j], target.Content[j+1])
					}
				}
				continue
			}
			if key.Kind != yamlv3.ScalarNode {
				return fmt.Errorf("mapping keys must be scalars")
			}
			id := key.Tag + "\x00" + key.Value
			if seen[id] {
				return fmt.Errorf("duplicate key %q at line %d", key.Value, key.Line)
			}
			seen[id] = true
			if err := resolve(value, dir, active, depth); err != nil {
				return err
			}
			appendPair(key, value)
		}
		node.Content = content
		return nil
	}
	for _, child := range node.Content {
		if err := resolve(child, dir, active, depth); err != nil {
			return err
		}
	}
	return nil
}

func filenames(node *yamlv3.Node) ([]string, error) {
	nodes := []*yamlv3.Node{node}
	if node.Kind == yamlv3.SequenceNode {
		nodes = node.Content
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("directive requires at least one filename")
	}
	names := make([]string, 0, len(nodes))
	for _, item := range nodes {
		if item.Kind != yamlv3.ScalarNode || item.Value == "" || (item.Tag != "!!str" && item.Tag != "!include") {
			return nil, fmt.Errorf("directive requires nonempty filename strings")
		}
		names = append(names, item.Value)
	}
	return names, nil
}

func combine(base, incoming *yamlv3.Node) error {
	if base.Kind != incoming.Kind {
		return fmt.Errorf("multi-include requires matching mappings or sequences")
	}
	switch base.Kind {
	case yamlv3.SequenceNode:
		base.Content = append(base.Content, incoming.Content...)
	case yamlv3.MappingNode:
		indices := make(map[string]int)
		for i := 0; i < len(base.Content); i += 2 {
			indices[base.Content[i].Tag+"\x00"+base.Content[i].Value] = i
		}
		for i := 0; i < len(incoming.Content); i += 2 {
			key := incoming.Content[i]
			id := key.Tag + "\x00" + key.Value
			if index, ok := indices[id]; ok {
				base.Content[index+1] = incoming.Content[i+1]
			} else {
				indices[id] = len(base.Content)
				base.Content = append(base.Content, key, incoming.Content[i+1])
			}
		}
	default:
		return fmt.Errorf("multi-include supports only mappings or sequences")
	}
	return nil
}
