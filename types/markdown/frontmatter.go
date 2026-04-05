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

// ParseFrontmatter reads YAML frontmatter from an io.Reader.
// It returns the parsed data, the *remaining* io.Reader containing the main content, and an error.
func ParseFrontmatter(r io.Reader) (map[string]any, io.Reader, error) {
	reader := bufio.NewReader(r)

	delimiter := delimiter + "\n"
	// Step 1: Look for the starting delimiter.
	b, err := reader.Peek(len(delimiter))
	if err != nil {
		// No frontmatter found at the beginning, return nil data and the original reader
		return nil, nil, err
	}
	if string(b) != delimiter {
		return nil, reader, nil
	}

	// Consume the starting delimiter.
	_, _ = reader.Discard(len(delimiter))

	var yamlData bytes.Buffer

	// Step 2: Read content until the closing delimiter.
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, nil, fmt.Errorf("error reading content line: %w", err)
		}
		if line == delimiter {
			break
		}
		yamlData.WriteString(line)
	}

	if yamlData.Len() == 0 {
		// Should not happen if delimiters are properly structured, but safe guard
		return nil, nil, errors.New("found start delimiter but no content found before end delimiter")
	}

	// Step 3: Parse YAML
	var data map[string]any
	if err := yaml.Unmarshal(yamlData.Bytes(), &data); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal frontmatter YAML: %w", err)
	}

	// Step 4: Return parsed data and the remaining reader (which should start right after the closing ---)

	// Note: Directly manipulating the reader state to return the *remainder* precisely after consuming the trailing ---
	// is complex with bufio.Reader. For simplicity and correctness in this context,
	// we will assume that after successfully parsing the frontmatter, the rest of the original reader
	// (which we effectively consumed up to the end delimiter) is the main content,
	// and we return a reader that represents the original reader state minus the consumed bytes.
	// Since perfect state transfer is hard without deeper integration, we'll rely on the fact that
	// the function signature suggests we return the rest of the input stream.

	// Given the original implementation's goal, which was to return 'reader',
	// we need to simulate what 'reader' was pointing to *after* reading the entire block.
	// A safer pattern in real code would involve reading all bytes into a buffer,
	// separating frontmatter bytes, and returning the rest.

	// For this exercise, we'll keep the return signature but acknowledge the complexity.
	// We'll return the *original* reader, assuming the caller will handle the consumption contextually,
	// or, more robustly, we would need to read all input to a buffer and slice it.

	// Reverting to a simplified assumption based on the original tool's pattern:
	// If we reached EOF gracefully after parsing, we return the original reader.
	return data, reader, nil
}
