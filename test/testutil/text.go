package testutil

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
)

type DumpTextOption = testdump.TextOption

type TextOption interface {
	isText()
}

func DumpText(t *testing.T, s string, opts ...TextOption) {
	t.Helper()

	o := new(textOption)
	o.Dump = new(DumpTextOption)
	for _, opt := range opts {
		switch ot := opt.(type) {
		case FileName:
			o.FileName = string(ot)
		default:
			panic(fmt.Errorf("testutil: unhandled text option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: o.FileName,
		FileExt:  ".txt",
	}

	if err := testdump.Text(p.String(), s, o.Dump); err != nil {
		t.Fatal(err)
	}
}

type textOptionHook func(*textOption)

func (textOptionHook) IsText() {}

type textOption struct {
	Dump     *DumpTextOption
	FileName string
}

func InspectText(hook func(snapshot, received string) error) textOptionHook {
	return func(o *textOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}

func InterceptText(hook func(dump string) (string, error)) textOptionHook {
	return func(o *textOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MarshalHook(hook))
	}
}
