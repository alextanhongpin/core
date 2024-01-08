package testutil

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/test/testdump"
)

type TextOption interface {
	isText()
}

func DumpText(t *testing.T, s string, opts ...TextOption) {
	t.Helper()

	var fileName string
	var hooks []testdump.Hook[string]
	for _, opt := range opts {
		switch o := opt.(type) {
		case FileName:
			fileName = string(o)
		case *textHookOption:
			hooks = append(hooks, o.hook)
		default:
			panic(fmt.Errorf("testutil: unhandled text option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: fileName,
		FileExt:  ".txt",
	}

	if err := testdump.Text(testdump.NewFile(p.String()), s); err != nil {
		t.Fatal(err)
	}
}

type textHookOption struct {
	hook testdump.Hook[string]
}

func (*textHookOption) isText() {}

func InspectText(hook func(snapshot, received string) error) *textHookOption {
	return &textHookOption{
		hook: testdump.CompareHook(hook),
	}
}

func InterceptText(hook func(dump string) (string, error)) *textHookOption {
	return &textHookOption{
		hook: testdump.MarshalHook(hook),
	}
}
