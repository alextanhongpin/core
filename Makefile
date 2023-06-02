gofiles := $(shell go list ./... | grep -v 'database')

cover:
	@go test -v -race -cover -coverprofile=cover.out $(gofiles)
	@go tool cover -html cover.out
