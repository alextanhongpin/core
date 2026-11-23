package markdown

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"

	"sigs.k8s.io/yaml"
)

const delimiter = "---"

// ErrUnterminated is returned by ParseFrontmatter when the opening
// delimiter has no closing delimiter.
var ErrUnterminated = errors.New("markdown: unterminated frontmatter")

// WriteFrontmatter writes meta as YAML between "---" lines.
func WriteFrontmatter(w io.Writer, meta any) error {
	b, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n%s%s\n", delimiter, b, delimiter)
	return err
}

// Write writes the frontmatter followed by content.
func Write(w io.Writer, meta any, content []byte) error {
	if err := WriteFrontmatter(w, meta); err != nil {
		return err
	}
	_, err := w.Write(content)
	return err
}

// WriteString writes the frontmatter followed by content.
func WriteString(w io.Writer, meta any, content string) error {
	return Write(w, meta, []byte(content))
}

// ParseFrontmatter decodes the YAML frontmatter at the start of r into T and
// returns a reader positioned at the body. Frontmatter starts with a "---"
// line and ends at the next "---" line; "\r\n" line endings are accepted.
// Without frontmatter it returns the zero T and a reader over all of r.
func ParseFrontmatter[T any](r io.Reader) (T, io.Reader, error) {
	var zero T
	br := bufio.NewReader(r)

	first, err := br.ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return zero, nil, err
	}
	if !isDelimiter(first) || errors.Is(err, io.EOF) {
		return zero, io.MultiReader(bytes.NewReader(first), br), nil
	}

	var data []byte
	for {
		line, err := br.ReadBytes('\n')
		if isDelimiter(line) {
			break
		}
		data = append(data, line...)
		if errors.Is(err, io.EOF) {
			return zero, nil, ErrUnterminated
		}
		if err != nil {
			return zero, nil, err
		}
	}

	var meta T
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return zero, nil, fmt.Errorf("markdown: decode frontmatter: %w", err)
	}
	return meta, br, nil
}

func isDelimiter(line []byte) bool {
	return string(bytes.TrimRight(line, "\r\n")) == delimiter
}
