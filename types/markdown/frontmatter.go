package markdown

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"

	"sigs.k8s.io/yaml"
)

var delimiter = "---"

// WriteFrontmatter writes frontmatter data to the writer in the format:
// ---
// key: value
// key2: value2
// ---
// The input map 'meta' should ideally be a map[string]any.
func WriteFrontmatter(w io.Writer, meta any) error {
	b, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n%s%s\n", delimiter, b, delimiter)
	return err
}

// Write writes both frontmatter and bytes content to the writer.
func Write(w io.Writer, meta any, content []byte) error {
	err := WriteFrontmatter(w, meta)
	if err != nil {
		return err
	}
	_, err = w.Write(content)
	return err
}

// WriteString writes both frontmatter and string content to the writer.
func WriteString(w io.Writer, meta any, content string) error {
	err := WriteFrontmatter(w, meta)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, content)
	return err
}

// ParseFrontmatter reads YAML frontmatter from an io.Reader.
// It returns the parsed data, the *remaining* io.Reader containing the main content, and an error.
func ParseFrontmatter[T any](r io.Reader) (T, io.Reader, error) {
	var zero T

	reader := bufio.NewReader(r)

	delimiter := []byte(delimiter + "\n")
	// Step 1: Look for the starting delimiter.
	b, err := reader.Peek(len(delimiter))
	if err != nil {
		return zero, nil, err
	}
	// No frontmatter found at the beginning, return nil data and the original reader
	if !bytes.Equal(b, delimiter) {
		return zero, reader, nil
	}

	// Consume the starting delimiter.
	_, _ = reader.Discard(len(delimiter))

	var data []byte
	// Step 2: Read content until the closing delimiter.
	for {
		b, err := reader.ReadBytes('\n')
		if err != nil {
			return zero, nil, fmt.Errorf("error reading content line: %w", err)
		}
		if bytes.Equal(b, delimiter) {
			break
		}
		data = append(data, b...)
	}

	if len(data) == 0 {
		// Should not happen if delimiters are properly structured, but safe guard
		return zero, nil, errors.New("found start delimiter but no content found before end delimiter")
	}

	// Step 3: Parse YAML
	var meta T
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return zero, nil, fmt.Errorf("failed to unmarshal frontmatter YAML: %w", err)
	}

	return meta, reader, nil
}
