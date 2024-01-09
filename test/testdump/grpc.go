package testdump

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alextanhongpin/core/grpc/grpcdump"
	"github.com/alextanhongpin/core/internal"
	"github.com/alextanhongpin/core/types/maputil"
	"github.com/google/go-cmp/cmp"
)

var ErrMetadataNotFound = errors.New("testdump: gRPC metadata not found")

type GRPCDump = grpcdump.Dump

type GRPCOption struct {
	Message  []cmp.Option
	Metadata []cmp.Option
	// TODO: Header and trailer
}

func GRPC(rw readerWriter, dump *GRPCDump, opt *GRPCOption, hooks ...Hook[*GRPCDump]) error {
	if opt == nil {
		opt = new(GRPCOption)
	}

	var s S[*GRPCDump] = &snapshot[*GRPCDump]{
		marshaler:   MarshalFunc[*GRPCDump](MarshalGRPC),
		unmarshaler: UnmarshalFunc[*GRPCDump](UnmarshalGRPC),
		comparer: &GRPCComparer{
			Message:  opt.Message,
			Metadata: opt.Metadata,
		},
	}

	s = Hooks[*GRPCDump](hooks).Apply(s)

	return Snapshot(rw, dump, s)
}

func MarshalGRPC(d *GRPCDump) ([]byte, error) {
	return d.AsText()
}

func UnmarshalGRPC(b []byte) (*GRPCDump, error) {
	d := new(GRPCDump)
	err := d.FromText(b)
	return d, err
}

type GRPCComparer struct {
	Message  []cmp.Option
	Metadata []cmp.Option
}

func (c GRPCComparer) Compare(snapshot, received *GRPCDump) error {
	x := snapshot
	y := received

	if err := internal.ANSIDiff(x.Addr, y.Addr); err != nil {
		return fmt.Errorf("Addr: %w", err)
	}

	if err := internal.ANSIDiff(x.FullMethod, y.FullMethod); err != nil {
		return fmt.Errorf("Full Method: %w", err)
	}

	if err := internal.ANSIDiff(x.Messages, y.Messages, c.Message...); err != nil {
		return fmt.Errorf("Message: %w", err)
	}

	if err := internal.ANSIDiff(x.Status, y.Status); err != nil {
		return fmt.Errorf("Status: %w", err)
	}

	if err := internal.ANSIDiff(x.Metadata, y.Metadata, c.Metadata...); err != nil {
		return fmt.Errorf("Metadata: %w", err)
	}

	if err := internal.ANSIDiff(x.Header, y.Header, c.Metadata...); err != nil {
		return fmt.Errorf("Header: %w", err)
	}

	if err := internal.ANSIDiff(x.Trailer, y.Trailer, c.Metadata...); err != nil {
		return fmt.Errorf("Trailer: %w", err)
	}

	return nil
}

func MaskMetadata(headers ...string) Hook[*GRPCDump] {
	type T = *GRPCDump

	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S: s,
			hook: func(t T) (T, error) {
				for _, h := range headers {
					var count int
					{
						v := t.Metadata.Get(h)
						m := make([]string, len(v))
						for i := 0; i < len(v); i++ {
							m[i] = maputil.MaskValue
						}
						t.Metadata.Set(h, m...)

						count += len(v)
					}

					{
						v := t.Trailer.Get(h)
						m := make([]string, len(v))
						for i := 0; i < len(v); i++ {
							m[i] = maputil.MaskValue
						}
						t.Trailer.Set(h, m...)

						count += len(v)
					}

					{
						v := t.Header.Get(h)
						m := make([]string, len(v))
						for i := 0; i < len(v); i++ {
							m[i] = maputil.MaskValue
						}
						t.Header.Set(h, m...)

						count += len(v)
					}

					if count == 0 {
						return nil, fmt.Errorf("%w for gRPC: %q", ErrMetadataNotFound, h)
					}
				}

				return t, nil
			},
		}
	}
}

func MaskMessage(fields ...string) Hook[*GRPCDump] {
	type T = *GRPCDump

	return func(s S[T]) S[T] {
		return &marshalHook[T]{
			S: s,
			hook: func(t T) (T, error) {
				for i, msg := range t.Messages {
					b, err := json.Marshal(msg)
					if err != nil {
						return t, err
					}

					b, err = maputil.MaskBytes(b, fields...)
					if err != nil {
						return nil, err
					}
					var msg grpcdump.Message
					if err := json.Unmarshal(b, &msg); err != nil {
						return t, err
					}

					t.Messages[i] = msg
				}

				return t, nil
			},
		}
	}
}
