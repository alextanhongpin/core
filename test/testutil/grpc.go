package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/grpc/grpcdump"
	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
	"github.com/google/go-cmp/cmp"
)

type GRPCDump = testdump.GRPCDump
type GRPCHook = testdump.Hook[*GRPCDump]

type GRPCOption interface {
	isGRPC()
}

func DumpGRPC(t *testing.T, ctx context.Context, opts ...GRPCOption) context.Context {
	t.Helper()
	ctx, flush := grpcdump.NewRecorder(ctx)

	var fileName string
	var hooks []testdump.Hook[*GRPCDump]
	grpcOpt := new(testdump.GRPCOption)

	for _, opt := range opts {
		switch o := opt.(type) {
		case FileName:
			fileName = string(o)
		case *grpcHookOption:
			hooks = append(hooks, o.hook)
		case *grpcCmpOption:
			grpcOpt.Message = append(grpcOpt.Message, o.message...)
			grpcOpt.Metadata = append(grpcOpt.Message, o.metadata...)
		default:
			panic(fmt.Errorf("testutil: unhandled gRPC option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: fileName,
		FileExt:  ".http",
	}

	t.Cleanup(func() {
		dump := flush()

		if err := testdump.GRPC(testdump.NewFile(p.String()), dump, grpcOpt, hooks...); err != nil {
			t.Fatal(err)
		}
	})

	return ctx
}

type grpcHookOption struct {
	hook testdump.Hook[*GRPCDump]
}

func (grpcHookOption) isGRPC() {}

type grpcCmpOption struct {
	metadata []cmp.Option
	message  []cmp.Option
}

func (grpcCmpOption) isGRPC() {}

func IgnoreMetadata(headers ...string) *grpcCmpOption {
	return &grpcCmpOption{
		metadata: []cmp.Option{internal.IgnoreMapEntries(headers...)},
	}
}

func IgnoreMessageFields(fields ...string) *grpcCmpOption {
	return &grpcCmpOption{
		message: []cmp.Option{internal.IgnoreMapEntries(fields...)},
	}
}

func MaskMetadata(headers ...string) *grpcHookOption {
	return &grpcHookOption{
		hook: testdump.MaskMetadata(headers...),
	}
}

func MaskMessage(fields ...string) *grpcHookOption {
	return &grpcHookOption{
		hook: testdump.MaskMessage(fields...),
	}
}

func InspectGRPC(hook func(snapshot, received *GRPCDump) error) *grpcHookOption {
	return &grpcHookOption{
		hook: testdump.CompareHook(hook),
	}
}

func InterceptGRPC(hook func(dump *GRPCDump) (*GRPCDump, error)) *grpcHookOption {
	return &grpcHookOption{
		hook: testdump.MarshalHook(hook),
	}
}
