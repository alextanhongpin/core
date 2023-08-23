package testdump_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
	"github.com/stretchr/testify/assert"
)

func TestText(t *testing.T) {
	fileName := fmt.Sprintf("testdata/%s.txt", t.Name())
	text := "What a wonderful world"

	if err := testdump.Text(testdump.NewFile(fileName), text, nil); err != nil {
		t.Fatal(err)
	}
}

func TestTextInMemory(t *testing.T) {
	im := testdump.NewInMemory()

	assert := assert.New(t)
	assert.Nil(testdump.Text(im, "foo", nil))
	assert.NotNil(testdump.Text(im, "bar", nil))

	err := testdump.Text(im, "bar", nil)
	diffErr, ok := testdump.AsDiffError(err)
	assert.True(ok)
	diffErr.SetColor(false)

	diff := diffErr.Error()
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "+") {
			assert.True(ok)
			assert.Contains(line, "bar")
		}

		if strings.HasPrefix(line, "-") {
			assert.True(ok)
			assert.Contains(line, "foo")
		}
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

	if err := testdump.Text(testdump.NewFile(fileName), text, &opt); err != nil {
		t.Fatal(err)
	}
}
