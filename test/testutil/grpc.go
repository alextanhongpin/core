package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/grpc/grpcdump"
	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
)

type GRPCDumpOption = testdump.GRPCOption
type GRPCDump = testdump.GRPCDump
type GRPCHook = testdump.GRPCHook

type GRPCOption interface {
	isGRPC()
}

func DumpGRPC(t *testing.T, ctx context.Context, opts ...GRPCOption) context.Context {
	t.Helper()
	ctx, flush := grpcdump.NewRecorder(ctx)

	o := new(grpcOption)
	o.Dump = new(GRPCDumpOption)
	for _, opt := range opts {
		switch ot := opt.(type) {
		case FileName:
			o.FileName = string(ot)
		case grpcOptionHook:
			ot(o)
		default:
			panic(fmt.Errorf("testutil: unhandled gRPC option: %#v", opt))
		}
	}

	p := Path{
		Dir:      "testdata",
		FilePath: t.Name(),
		FileName: o.FileName,
		FileExt:  ".http",
	}
	fileName := p.String()

	t.Cleanup(func() {
		dump := flush()
		if err := testdump.GRPC(fileName, dump, o.Dump); err != nil {
			t.Fatal(err)
		}
	})

	return ctx
}

type grpcOptionHook func(o *grpcOption)

func (grpcOptionHook) isGRPC() {}

type grpcOption struct {
	Dump     *GRPCDumpOption
	FileName string
}

func IgnoreMetadata(headers ...string) grpcOptionHook {
	return func(o *grpcOption) {
		o.Dump.Metadata = append(o.Dump.Metadata,
			internal.IgnoreMapEntries(headers...),
		)
	}
}

func IgnoreMessageFields(fields ...string) grpcOptionHook {
	return func(o *grpcOption) {
		o.Dump.Message = append(o.Dump.Message,
			internal.IgnoreMapEntries(fields...),
		)
	}
}

func MaskMetadata(headers ...string) grpcOptionHook {
	return func(o *grpcOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskMetadata(headers...),
		)
	}
}

func MaskMessage(fields ...string) grpcOptionHook {
	return func(o *grpcOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskMessage(fields...),
		)
	}
}

func InspectGRPC(hook func(snapshot, received *GRPCDump) error) grpcOptionHook {
	return func(o *grpcOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}

func InterceptGRPC(hook func(dump *GRPCDump) (*GRPCDump, error)) grpcOptionHook {
	return func(o *grpcOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MarshalHook(hook))
	}
}
