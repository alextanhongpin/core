package ratelimit

import (
	"context"
	_ "embed"
	"fmt"
	"reflect"
	"strings"

	redis "github.com/redis/go-redis/v9"
)

//go:embed code.lua
var code string

// LibraryName is the name of the Redis function library installed by Setup.
// It provides the functions rl_fixed_window and rl_gcra.
const LibraryName = "ratelimit"

// Setup installs the rate limiter's Lua functions into Redis using
// FUNCTION LOAD REPLACE. It must succeed before any limiter is used.
//
// Setup is idempotent: calling it at every application start is safe and
// recommended, because it also upgrades the functions after a new release
// of this package. Function names and arguments are kept backward compatible,
// so instances running an older version keep working during a rolling deploy.
//
// For a *redis.ClusterClient the library is loaded on every master, and for a
// *redis.Ring on every shard. Redis persists and replicates functions like
// data, but a fresh server without persistence starts empty, so run Setup on
// startup rather than once by hand.
func Setup(ctx context.Context, client redis.UniversalClient) error {
	if isNil(client) {
		return fmt.Errorf("ratelimit: nil client")
	}
	load := func(ctx context.Context, c *redis.Client) error {
		return c.FunctionLoadReplace(ctx, code).Err()
	}
	var err error
	switch c := client.(type) {
	case *redis.ClusterClient:
		err = c.ForEachMaster(ctx, load)
	case *redis.Ring:
		err = c.ForEachShard(ctx, load)
	default:
		err = client.FunctionLoadReplace(ctx, code).Err()
	}
	if err != nil {
		return fmt.Errorf("ratelimit: setup: %w", err)
	}
	return nil
}

// wrapErr adds a hint to the error Redis returns when Setup was not run.
func wrapErr(err error) error {
	if err != nil && strings.Contains(err.Error(), "Function not found") {
		return fmt.Errorf("%w: %w", ErrNotSetup, err)
	}
	return err
}

func isNil(client redis.UniversalClient) bool {
	if client == nil {
		return true
	}
	v := reflect.ValueOf(client)
	return v.Kind() == reflect.Pointer && v.IsNil()
}
