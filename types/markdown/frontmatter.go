package markdown

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"

	"sigs.k8s.io/yaml"
)

func WriteFrontmatter(fm any, w io.Writer) error {
	b, err := yaml.Marshal(fm)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "---\n%s---\n", b)
	return err
}

func ParseFrontmatter(r io.Reader) (map[string]any, io.Reader, error) {
	reader := bufio.NewReader(r)
	isDelim := func() (bool, error) {
		b, err := reader.Peek(4)
		if err != nil {
			return false, err
		}
		if string(b) != "---\n" {
			return false, nil
		}
		_, _ = reader.Discard(4)
		return true, nil
	}

	ok, err := isDelim()
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, reader, nil
	}

	bb := new(bytes.Buffer)
	for {
		ok, err := isDelim()
		if ok || errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		s, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read string error: %w", err)
		}
		bb.WriteString(s)
	}

	var a map[string]any
	if b := bb.Bytes(); len(b) > 0 {
		err = yaml.Unmarshal(b, &a)
		if err != nil {
			return nil, nil, err
		}
	}

	return a, reader, nil
}
