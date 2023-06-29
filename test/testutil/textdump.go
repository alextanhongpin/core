package testutil

import (
	"testing"
)

func DumpTextFile(fileName, content string, opts ...TextOption) error {
	type dumpAndCompare struct {
		dumper
		comparer
	}

	dnc := dumpAndCompare{
		dumper:   TextDumper(content),
		comparer: TextComparer(""),
	}

	return Dump(fileName, dnc)
}

// DumpText dumps a type as json.
func DumpText(t *testing.T, content string, opts ...TextOption) string {
	t.Helper()

	p := NewTextPath(opts...)
	if p.FilePath == "" {
		p.FilePath = t.Name()
	}

	fileName := p.String()

	if err := DumpTextFile(fileName, content, opts...); err != nil {
		t.Fatal(err)
	}

	return fileName
}

type TextDumper string

func (d TextDumper) Dump() ([]byte, error) {
	return []byte(string(d)), nil
}

type TextComparer string

func (d TextComparer) Compare(snapshot, received []byte) error {
	return ansiDiff(snapshot, received)
}
