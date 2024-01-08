package testdump_test

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
	"github.com/stretchr/testify/assert"
)

func TestText(t *testing.T) {
	fileName := fmt.Sprintf("testdata/%s.txt", t.Name())
	text := "What a wonderful world"

	if err := testdump.Text(testdump.NewFile(fileName), text); err != nil {
		t.Fatal(err)
	}
}

func TestTextInMemory(t *testing.T) {
	im := testdump.NewInMemory()

	assert := assert.New(t)
	assert.Nil(testdump.Text(im, "foo"))
	assert.NotNil(testdump.Text(im, "bar"))

	err := testdump.Text(im, "bar")
	diffErr, ok := testdump.AsDiffError(err)
	assert.True(ok)
	diffErr.SetColor(false)

	testdump.Text(testdump.NewFile(fmt.Sprintf("testdata/%s.txt", t.Name())), diffErr.Text())
}
