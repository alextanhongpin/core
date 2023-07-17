package testutil

import (
	"context"
	"testing"

	"github.com/alextanhongpin/core/grpc/grpcdump"
	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/test/testdump"
)

type GRPCDumpOption = testdump.GRPCOption
type GRPCDump = testdump.GRPCDump
type GRPCHook = testdump.GRPCHook

type GRPCOption func(o *GrpcOption)

type GrpcOption struct {
	Dump     *GRPCDumpOption
	FileName string
}

func DumpGRPC(t *testing.T, ctx context.Context, opts ...GRPCOption) context.Context {
	t.Helper()
	ctx, flush := grpcdump.NewRecorder(ctx)

	o := new(GrpcOption)
	o.Dump = new(GRPCDumpOption)
	for _, opt := range opts {
		opt(o)
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

func IgnoreMetadata(headers ...string) GRPCOption {
	return func(o *GrpcOption) {
		o.Dump.Metadata = append(o.Dump.Metadata,
			internal.IgnoreMapEntries(headers...),
		)
	}
}

func IgnoreMessageFields(fields ...string) GRPCOption {
	return func(o *GrpcOption) {
		o.Dump.Message = append(o.Dump.Message,
			internal.IgnoreMapEntries(fields...),
		)
	}
}

func MaskMetadata(headers ...string) GRPCOption {
	return func(o *GrpcOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskMetadata(headers...),
		)
	}
}

func MaskMessage(fields ...string) GRPCOption {
	return func(o *GrpcOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MaskMessage(fields...),
		)
	}
}

func InspectGRPC(hook func(snapshot, received *GRPCDump) error) GRPCOption {
	return func(o *GrpcOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.CompareHook(hook))
	}
}

func InterceptGRPC(hook func(dump *GRPCDump) (*GRPCDump, error)) GRPCOption {
	return func(o *GrpcOption) {
		o.Dump.Hooks = append(o.Dump.Hooks,
			testdump.MarshalHook(hook))
	}
}

func GRPCFileName(name string) GRPCOption {
	return func(o *GrpcOption) {
		o.FileName = name
	}
}
