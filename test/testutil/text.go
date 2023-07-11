package testutil

import (
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
)

type DumpTextOption = testdump.TextOption

type TextOption func(*TxtOption)

type TxtOption struct {
	Dump     *DumpTextOption
	FileName string
}

func DumpText(t *testing.T, s string, opts ...TextOption) {
	t.Helper()

	o := new(TxtOption)
	o.Dump = new(DumpTextOption)

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

func TextFileName(name string) TextOption {
	return func(o *TxtOption) {
		o.FileName = name
	}
}

func InspectText(hook func(snapshot, received string) error) TextOption {
	return func(o *TxtOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}

func InterceptText(hook func(dump string) (string, error)) TextOption {
	return func(o *TxtOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MarshalHook(hook))
	}
}
