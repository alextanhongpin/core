package grpctest

import (
	"context"
	"errors"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

const addr = "bufnet"

func DialContext(ctx context.Context, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts, grpc.WithContextDialer(bufDialer))
	conn, err := grpc.DialContext(ctx, addr, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func ListenAndServe(fn func(*grpc.Server), opts ...grpc.ServerOption) func() {
	lis = bufconn.Listen(bufSize)

	srv := grpc.NewServer(opts...)
	fn(srv)

	done := make(chan bool)

	go func() {
		defer close(done)
		if err := srv.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			panic(err)
		}
	}()

	return func() {
		srv.Stop()
		lis.Close()
		<-done
	}
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}
