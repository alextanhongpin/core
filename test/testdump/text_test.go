package testdump_test

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
)

func TestText(t *testing.T) {
	fileName := fmt.Sprintf("testdata/%s.txt", t.Name())
	text := "What a wonderful world"

	if err := testdump.Text(fileName, text, nil); err != nil {
		t.Fatal(err)
	}
}

func TestTextHook(t *testing.T) {
	fileName := fmt.Sprintf("testdata/%s.txt", t.Name())
	text := "hello world"

	opt := testdump.TextOption{
		Hooks: []testdump.Hook[string]{
			testdump.MarshalHook(func(str string) (string, error) {
				return fmt.Sprintf("%s %s", str, "1..."), nil
			}),
			testdump.MarshalHook(func(str string) (string, error) {
				return fmt.Sprintf("%s %s", str, "2..."), nil
			}),
		},
	}

	if err := testdump.Text(fileName, text, &opt); err != nil {
		t.Fatal(err)
	}
}
