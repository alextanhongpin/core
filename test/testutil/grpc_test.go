package testutil_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	pb "github.com/alextanhongpin/core/grpc/examples/helloworld/v1"
	"github.com/alextanhongpin/core/grpc/grpcdump"
	"github.com/alextanhongpin/core/grpc/grpctest"
	"github.com/alextanhongpin/core/test/testutil"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestMain(m *testing.M) {
	stop := grpctest.ListenAndServe(func(srv *grpc.Server) {
		// Register your server here.
		pb.RegisterGreeterServiceServer(srv, &server{})
	},
		// Setup grpcdump on the server side.
		// Also set on the client side, but only the `WithUnaryInterceptor`.
		grpcdump.UnaryInterceptor(),
		grpcdump.StreamInterceptor(),
	)

	code := m.Run()
	stop()
	os.Exit(code)
}

func TestGRPCClientStreaming(t *testing.T) {
	ctx := context.Background()
	conn := grpcDialContext(t, ctx)

	// Create a new client.
	client := pb.NewGreeterServiceClient(conn)

	// Create a new recorder.
	ctx = testutil.DumpGRPC(t, ctx)
	ctx = metadata.AppendToOutgoingContext(ctx,
		"md-val", "md-val",
		"md-val-bin", "md-val-bin",
	)

	assert := assert.New(t)
	stream, err := client.RecordGreetings(ctx)
	assert.Nil(err)

	// Send 5 greetings.
	n := 5
	for i := 0; i < n; i++ {
		err := stream.Send(&pb.RecordGreetingsRequest{
			Message: "hi sir",
		})
		assert.Nil(err)
	}

	reply, err := stream.CloseAndRecv()
	assert.Nil(err)
	assert.Equal(reply.GetCount(), int64(n))
}

func TestGRPCBidirectionalStreaming(t *testing.T) {
	ctx := context.Background()
	conn := grpcDialContext(t, ctx)

	// Create a new client.
	client := pb.NewGreeterServiceClient(conn)

	// Create a new recorder.
	ctx = testutil.DumpGRPC(t, ctx)
	ctx = metadata.AppendToOutgoingContext(ctx,
		"md-val", "md-val",
		"md-val-bin", "md-val-bin",
	)

	assert := assert.New(t)
	stream, err := client.Chat(ctx)
	assert.Nil(err)

	done := make(chan bool)

	go func() {
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				close(done)
				return
			}
			assert.Nil(err)
		}
	}()

	for _, msg := range []string{"foo", "bar"} {
		err := stream.Send(&pb.ChatRequest{
			Message: msg,
		})
		assert.Nil(err)
	}
	stream.CloseSend()

	<-done
}

func TestGRPCServerStreaming(t *testing.T) {
	ctx := context.Background()
	conn := grpcDialContext(t, ctx)

	// Create a new client.
	client := pb.NewGreeterServiceClient(conn)

	// Create a new recorder.
	ctx = testutil.DumpGRPC(t, ctx)

	ctx = metadata.AppendToOutgoingContext(ctx,
		"md-val", "md-val",
		"md-val-bin", "md-val-bin",
	)

	assert := assert.New(t)
	stream, err := client.ListGreetings(ctx, &pb.ListGreetingsRequest{
		Count: 5,
	})
	assert.Nil(err)

	done := make(chan bool)

	go func() {
		defer close(done)
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				break
			}

			assert.Nil(err)
		}
	}()

	<-done
}

func TestGRPCUnary(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		conn := grpcDialContext(t, ctx)

		// Send token.
		md := metadata.New(map[string]string{
			"authorization": "xyz",
		})

		// This will override all other context, so call this first.
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Create a new client.
		client := pb.NewGreeterServiceClient(conn)

		// Create a new recorder.
		ctx = testutil.DumpGRPC(t, ctx)

		ctx = metadata.AppendToOutgoingContext(ctx,
			"md-val", "md-val",
			"md-val-bin", "md-val-bin",
		)

		_, err := client.SayHello(ctx, &pb.SayHelloRequest{
			Name: "John Doe",
		})
		assert.Nil(t, err)
	})

	t.Run("unauthorized", func(t *testing.T) {
		conn := grpcDialContext(t, ctx)

		// Send token.
		md := metadata.New(map[string]string{
			"authorization": "abc",
		})

		// This will override all other context, so call this first.
		ctx = metadata.NewOutgoingContext(ctx, md)

		// Create a new client.
		client := pb.NewGreeterServiceClient(conn)

		// Create a new recorder.
		ctx = testutil.DumpGRPC(t, ctx)

		ctx = metadata.AppendToOutgoingContext(ctx,
			"md-val", "md-val",
			"md-val-bin", "md-val-bin",
		)

		// Anything linked to this variable will fetch response headers.
		_, err := client.SayHello(ctx, &pb.SayHelloRequest{
			Name: "John Doe",
		})
		assert.NotNil(t, err)
	})
}

type server struct {
	pb.UnimplementedGreeterServiceServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.SayHelloRequest) (*pb.SayHelloResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no token present")
	}

	token := md.Get("authorization")[0]
	if token != "xyz" {
		return nil, status.Error(codes.Unauthenticated, "token expired")
	}

	ctx = metadata.NewOutgoingContext(ctx, nil)
	ctx = metadata.AppendToOutgoingContext(ctx,
		"header-key", "header-val",
		"header-key-bin", "header-val-bin",
	)
	header, _ := metadata.FromOutgoingContext(ctx)

	// For unary.
	if err := grpc.SendHeader(ctx, header); err != nil {
		return nil, status.Errorf(codes.Internal, "unable to send gRPC header: %v", err)
	}

	trailer := metadata.New(map[string]string{
		"trailer-key":     "trailer-val",
		"trailer-key-bin": "trailer-val-bin",
	})

	// For unary.
	if err := grpc.SetTrailer(ctx, trailer); err != nil {
		return nil, status.Errorf(codes.Internal, "unable to send gRPC trailer: %v", err)
	}

	return &pb.SayHelloResponse{Message: "Hello " + in.GetName()}, nil
}

func (s *server) RecordGreetings(stream pb.GreeterService_RecordGreetingsServer) error {
	ctx := stream.Context()
	ctx = metadata.AppendToOutgoingContext(ctx,
		"header-key", "header-val",
		"header-key-bin", "header-val-bin",
	)

	if header, ok := metadata.FromOutgoingContext(ctx); ok {
		// For stream.
		if err := stream.SendHeader(header); err != nil {
			return status.Errorf(codes.Internal, "unable to send gRPC header: %v", err)
		}
	}

	trailer := metadata.New(map[string]string{
		"trailer-key":     "trailer-val",
		"trailer-key-bin": "trailer-val-bin",
	})

	// For unary.
	stream.SetTrailer(trailer)

	var msgs []string
	for {
		msg, err := stream.Recv()
		if err == io.EOF {

			return stream.SendAndClose(&pb.RecordGreetingsResponse{
				Count: int64(len(msgs)),
			})
		}
		if err != nil {
			return err
		}
		msgs = append(msgs, msg.GetMessage())
	}
}

func (s *server) ListGreetings(in *pb.ListGreetingsRequest, stream pb.GreeterService_ListGreetingsServer) error {
	ctx := stream.Context()
	ctx = metadata.AppendToOutgoingContext(ctx,
		"header-key", "header-val",
		"header-key-bin", "header-val-bin",
	)

	header, _ := metadata.FromOutgoingContext(ctx)

	// For stream.
	if err := stream.SendHeader(header); err != nil {
		return status.Errorf(codes.Internal, "unable to send gRPC header: %v", err)
	}

	n := in.GetCount()
	for i := 0; i < int(n); i++ {
		err := stream.Send(&pb.ListGreetingsResponse{
			Message: fmt.Sprintf("hi sir (%d)", i+1),
		})
		if err != nil {
			return err
		}
	}

	trailer := metadata.New(map[string]string{
		"trailer-key":     "trailer-val",
		"trailer-key-bin": "trailer-val-bin",
	})

	// For stream.
	stream.SetTrailer(trailer)

	return nil
}

func (s *server) Chat(stream pb.GreeterService_ChatServer) error {
	ctx := stream.Context()
	ctx = metadata.AppendToOutgoingContext(ctx,
		"header-key", "header-val",
		"header-key-bin", "header-val-bin",
	)

	header, _ := metadata.FromOutgoingContext(ctx)
	if err := stream.SendHeader(header); err != nil {
		return status.Errorf(codes.Internal, "unable to send gRPC header: %v", err)
	}

	trailer := metadata.New(map[string]string{
		"trailer-key":     "trailer-val",
		"trailer-key-bin": "trailer-val-bin",
	})
	stream.SetTrailer(trailer)

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if err := stream.Send(&pb.ChatResponse{
			Message: "REPLY: " + in.GetMessage(),
		}); err != nil {
			return err
		}
	}
}

func grpcDialContext(t *testing.T, ctx context.Context) *grpc.ClientConn {
	conn, err := grpctest.DialContext(ctx,
		grpc.WithInsecure(),
		// Setup grpcdump on the client side.
		grpcdump.WithUnaryInterceptor(),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		conn.Close()
	})

	return conn
}
