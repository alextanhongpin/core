// Package yaml loads YAML with relative !include and ordered "!merge" directives.
package yaml

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	yamlv3 "gopkg.in/yaml.v3"
)

const (
	maxDepth = 100  // nested includes
	maxFiles = 1000 // files loaded in total, including repeats
)

// ErrPathEscape is returned for a directive whose path is absolute or leaves
// the root directory.
var ErrPathEscape = errors.New("include path escapes the root directory")

// LoadExtendedYAML loads filename and decodes it into out. Includes may only
// reach files in filename's directory and below (symlinks included); use
// Load with a wider fs.FS to include from parent directories.
func LoadExtendedYAML(filename string, out any) error {
	root, err := os.OpenRoot(filepath.Dir(filename))
	if err != nil {
		return err
	}
	defer root.Close()
	return Load(root.FS(), filepath.Base(filename), out)
}

// Load loads name from fsys, resolves directives and decodes the result into
// out. Directive paths are slash-separated and relative to the file that
// contains them; they must stay within fsys. Merges are shallow and processed
// in source order, with later values replacing earlier values.
func Load(fsys fs.FS, name string, out any) error {
	l := &loader{fsys: fsys, active: make(map[string]bool)}
	node, err := l.load(name, 0)
	if err != nil {
		return err
	}
	if err := node.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}
	return nil
}

type loader struct {
	fsys   fs.FS
	active map[string]bool // files on the current include path
	loaded int
}

func (l *loader) load(name string, depth int) (*yamlv3.Node, error) {
	if depth >= maxDepth {
		return nil, fmt.Errorf("include depth exceeds %d at %s", maxDepth, name)
	}
	if l.loaded++; l.loaded > maxFiles {
		return nil, fmt.Errorf("more than %d files loaded at %s", maxFiles, name)
	}
	if !fs.ValidPath(name) {
		return nil, fmt.Errorf("%w: %s", ErrPathEscape, name)
	}
	if l.active[name] {
		return nil, fmt.Errorf("include cycle at %s", name)
	}
	l.active[name] = true
	defer delete(l.active, name)

	f, err := l.fsys.Open(name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	defer f.Close()

	decoder := yamlv3.NewDecoder(f)
	var doc yamlv3.Node
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	var extra yamlv3.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("%s: multiple YAML documents are unsupported", name)
		}
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("%s: empty YAML document", name)
	}
	if err := l.resolve(doc.Content[0], path.Dir(name), depth); err != nil {
		return nil, fmt.Errorf("resolve %s: %w", name, err)
	}
	return doc.Content[0], nil
}

// include loads name relative to dir.
func (l *loader) include(dir, name string, depth int) (*yamlv3.Node, error) {
	if path.IsAbs(name) || filepath.IsAbs(name) {
		return nil, fmt.Errorf("%w: %s", ErrPathEscape, name)
	}
	return l.load(path.Join(dir, name), depth+1)
}

func (l *loader) resolve(node *yamlv3.Node, dir string, depth int) error {
	if node.Tag == "!include" {
		names, err := filenames(node)
		if err != nil {
			return err
		}
		var merged *yamlv3.Node
		for _, name := range names {
			target, err := l.include(dir, name, depth)
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
					target, err := l.include(dir, name, depth)
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
			if err := l.resolve(value, dir, depth); err != nil {
				return err
			}
			appendPair(key, value)
		}
		node.Content = content
		return nil
	}
	for _, child := range node.Content {
		if err := l.resolve(child, dir, depth); err != nil {
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
