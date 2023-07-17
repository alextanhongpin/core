package grpcdump

import (
	"context"
	"fmt"
	"net"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TODO: Separate grpctest and grpcdump
// Update unary client to accept calloptions to extract header.
// Add metadata and trailers

const OriginServer = "server"
const OriginClient = "client"

const headerTestID = "x-test-id"

// NOTE: hackish implementation to extract the dump from the grpc server.
var testIds = make(map[string]*Dump)

// NewRecorder generates a new unique id for the request, and propagates it
// from the client request to the server.
// The request/response will then be dumped from the server and set to the
// global map with this id.
// The client can then retrieve the dump using the same id.
// The id is automatically cleaned up after the test is done.
func NewRecorder(ctx context.Context) (context.Context, func() *Dump) {
	// Generate a new unique id per test.
	id := uuid.New().String()

	ctx = metadata.AppendToOutgoingContext(ctx, headerTestID, id)

	return ctx, func() *Dump {
		dump := testIds[id]
		delete(testIds, id)

		return dump
	}
}

type Message struct {
	Origin  string `json:"origin"` // server or client
	Name    string `json:"name"`
	Message any    `json:"message"`
}

type serverStreamWrapper struct {
	grpc.ServerStream
	header   metadata.MD
	messages []Message
	trailer  metadata.MD
}

func (s *serverStreamWrapper) SetTrailer(md metadata.MD) {
	s.ServerStream.SetTrailer(md)

	s.trailer = metadata.Join(s.trailer, md)
}

func (s *serverStreamWrapper) SendHeader(md metadata.MD) error {
	if err := s.ServerStream.SendHeader(md); err != nil {
		return err
	}
	s.header = metadata.Join(s.header, md)

	return nil
}

func (s *serverStreamWrapper) SetHeader(md metadata.MD) error {
	if err := s.ServerStream.SetHeader(md); err != nil {
		return err
	}

	s.header = metadata.Join(s.header, md)

	return nil
}

func (s *serverStreamWrapper) SendMsg(m interface{}) error {
	if err := s.ServerStream.SendMsg(m); err != nil {
		return err
	}

	s.messages = append(s.messages, origin(OriginServer, m))

	return nil
}

func origin(origin string, v any) Message {
	msg, ok := v.(interface {
		ProtoReflect() protoreflect.Message
	})
	if !ok {
		panic("message is not valid")
	}

	return Message{
		Origin:  origin,
		Name:    fmt.Sprint(msg.ProtoReflect().Descriptor().FullName().Name()),
		Message: v,
	}
}

func (s *serverStreamWrapper) RecvMsg(m interface{}) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}

	s.messages = append(s.messages, origin(OriginClient, m))

	return nil
}

func StreamInterceptor() grpc.ServerOption {
	return grpc.StreamInterceptor(
		func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

			ctx := stream.Context()
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				panic("grpcdump: no metadata from incoming context")
			}

			// Extract the test-id from the header.
			// We do not want to log this, so delete it from the
			// existing header.
			id := md.Get(headerTestID)[0]
			md.Delete(headerTestID)

			w := &serverStreamWrapper{ServerStream: stream}
			err := handler(srv, w)

			testIds[id] = &Dump{
				Addr:       addrFromContext(ctx),
				FullMethod: info.FullMethod,
				Metadata:   md,
				Messages:   w.messages,
				Trailer:    w.trailer,
				Header:     w.header,
				Status:     NewStatus(err),
			}

			return err
		},
	)
}

func UnaryInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(
		func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				panic("grpcdump: no metadata from incoming context")
			}

			// Extract the test-id from the header.
			// We do not want to log this, so delete it from the
			// existing header.
			id := md.Get(headerTestID)[0]
			md.Delete(headerTestID)

			res, err := handler(ctx, req)
			messages := []Message{origin(OriginClient, req)}

			if err == nil {
				messages = append(messages, origin(OriginServer, res))
			}

			testIds[id] = &Dump{
				Addr:       addrFromContext(ctx),
				FullMethod: info.FullMethod,
				Metadata:   md,
				Messages:   messages,
				Status:     NewStatus(err),
			}

			return res, err
		},
	)
}

func WithUnaryInterceptor() grpc.DialOption {
	return grpc.WithUnaryInterceptor(grpc.UnaryClientInterceptor(
		func(ctx context.Context, method string, req, res interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				panic("grpcdump: missing test header id")
			}

			testID := md.Get(headerTestID)[0]

			ctx = metadata.NewOutgoingContext(ctx, md)

			var header, trailer metadata.MD
			opts = append(opts, grpc.Header(&header), grpc.Trailer(&trailer))

			if err := invoker(ctx, method, req, res, cc, opts...); err != nil {
				return err
			}

			header.Delete(headerTestID)

			testIds[testID].Trailer = trailer
			testIds[testID].Header = header

			return nil
		},
	))
}

func addrFromContext(ctx context.Context) string {
	var addr string
	if pr, ok := peer.FromContext(ctx); ok {
		if tcpAddr, ok := pr.Addr.(*net.TCPAddr); ok {
			addr = tcpAddr.IP.String()
		} else {
			addr = pr.Addr.String()
		}
	}
	return addr
}
