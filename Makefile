# The GODEBUG option is to allow the gRPC test that is configured with the
# certificate to pass the test.

gofiles := $(shell go list ./... | grep -v 'database')
gotest := GODEBUG=x509sha1=1 gotest

cover:
	@$(gotest) -v -failfast -race -cover -covermode=atomic -coverprofile=cover.out $(flag) $(gofiles)
	@go tool cover -html cover.out


install:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2



compile:
	@protoc \
		--go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    grpc/examples/helloworld/v1/helloworld.proto
