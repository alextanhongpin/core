package testutil_test

import (
	"testing"

	"github.com/alextanhongpin/core/test/testutil"
)

func TestTextDump(t *testing.T) {
	testutil.DumpText(t, "hello world")
}
