package testdump_test

import (
	"fmt"
	"testing"

	"github.com/alextanhongpin/core/grpc/grpcdump"
	"github.com/alextanhongpin/core/test/testdump"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func TestGRPC(t *testing.T) {
	dump := &grpcdump.Dump{
		Addr:       "bufconn",
		FullMethod: "/helloworld.v1.GreeterService/Chat",
		Status: &grpcdump.Status{
			Code:    codes.Unauthenticated.String(),
			Message: "not authenticated",
		},
		Metadata: metadata.New(map[string]string{
			"md-key":     "md-val",
			"md-key-bin": "md-val-bin",
		}),
		Header: metadata.New(map[string]string{
			"header-key":     "header-val",
			"header-key-bin": "header-val-bin",
		}),
		Trailer: metadata.New(map[string]string{
			"trailer-key":     "trailer-val",
			"trailer-key-bin": "trailer-val-bin",
		}),
		Messages: []grpcdump.Message{
			{
				Origin: grpcdump.OriginClient,
				Message: map[string]any{
					"msg": "Hello",
				},
				Name: "Message",
			},
			{
				Origin: grpcdump.OriginServer,
				Message: map[string]any{
					"msg": "Hi",
				},
				Name: "Message",
			},
		},
	}

	fileName := fmt.Sprintf("testdata/%s.http", t.Name())

	if err := testdump.GRPC(fileName, dump, &testdump.GRPCOption{}); err != nil {
		t.Fatal(err)
	}
}
