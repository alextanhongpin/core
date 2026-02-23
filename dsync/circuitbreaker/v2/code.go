package v2

import (
	_ "embed"

	"context"

	redis "github.com/redis/go-redis/v9"
)

//go:embed code.lua
var code string

func Setup(ctx context.Context, client *redis.Client) error {
	return client.FunctionLoadReplace(ctx, code).Err()
}
