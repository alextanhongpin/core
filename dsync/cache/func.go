package cache

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/redis/go-redis/v9/helper"
	"github.com/zeebo/xxh3"
)

// FuncConfig configures [Func] and [Idempotent].
type FuncConfig[K any] struct {
	// Codec encodes cached results. Optional; defaults to [JSONCodec].
	Codec Codec

	// KeyFn derives the Redis key from the request. Required. Include a
	// prefix, e.g. "user:" + id, to keep keys unique per function.
	KeyFn func(ctx context.Context, req K) (string, error)

	// Lock coordinates fills across processes. Required.
	Lock *Lock

	// Wait is how long a caller waits while another process runs fn for the
	// same key. Default 0: return ErrLocked immediately. See
	// [LoadOrCreateConfig].Wait.
	Wait time.Duration

	// LeaseTTL is the lease TTL while fn runs. Default 10s.
	LeaseTTL time.Duration

	// RefreshRatio enables lease renewal for long-running fn. Default 0 (off).
	RefreshRatio float32
}

func (c FuncConfig[K]) lease(create func(context.Context, string) ([]byte, time.Duration, error)) LoadOrCreateConfig[[]byte] {
	return LoadOrCreateConfig[[]byte]{
		Create:       create,
		Lock:         c.LeaseTTL,
		Wait:         c.Wait,
		RefreshRatio: c.RefreshRatio,
	}
}

// Func wraps fn with a Redis-backed read-through cache. The returned function
// computes the key with cfg.KeyFn and returns the cached result if present
// (loaded=true). Otherwise it calls fn under a [Lock] lease, caches the result
// with the TTL fn returns and returns it (loaded=false). Errors from fn are
// returned unchanged and are not cached.
func Func[K, V any](fn func(ctx context.Context, req K) (V, time.Duration, error), cfg FuncConfig[K]) (func(ctx context.Context, req K) (V, bool, error), error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if fn == nil {
		return nil, errors.New("cache: function is required")
	}
	return func(ctx context.Context, args K) (V, bool, error) {
		var zero V
		key, err := cfg.KeyFn(ctx, args)
		if err != nil {
			return zero, false, err
		}

		curr, loaded, err := cfg.Lock.LoadOrCreate(ctx, key, cfg.lease(func(ctx context.Context, key string) ([]byte, time.Duration, error) {
			v, ttl, err := fn(ctx, args)
			if err != nil {
				return nil, 0, err
			}
			bb := new(bytes.Buffer)
			err = cfg.Codec.NewEncoder(bb).Encode(v)
			if err != nil {
				return nil, 0, err
			}
			return bb.Bytes(), ttl, nil
		}))
		if err != nil {
			return zero, false, err
		}
		var v V
		err = cfg.Codec.NewDecoder(bytes.NewBuffer(curr)).Decode(&v)
		if err != nil {
			return zero, false, err
		}

		return v, loaded, nil
	}, nil
}

// Idempotent is like [Func] but also stores the request next to the response.
// A later call with the same key but a different request (compared by its
// canonical JSON form) returns [ErrConflict] instead of the cached response.
// Use it for idempotency keys: repeat requests get the first response, and a
// key reused for a different request is rejected.
func Idempotent[K, V any](fn func(ctx context.Context, req K) (V, time.Duration, error), cfg FuncConfig[K]) (func(ctx context.Context, req K) (V, bool, error), error) {
	cfg = cfg.WithDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if fn == nil {
		return nil, errors.New("cache: function is required")
	}
	type dto struct {
		Request  K `json:"request"`
		Response V `json:"response"`
	}

	return func(ctx context.Context, req K) (V, bool, error) {
		var zero V
		key, err := cfg.KeyFn(ctx, req)
		if err != nil {
			return zero, false, err
		}
		b, loaded, err := cfg.Lock.LoadOrCreate(ctx, key, cfg.lease(func(ctx context.Context, key string) ([]byte, time.Duration, error) {
			v, ttl, err := fn(ctx, req)
			if err != nil {
				return nil, 0, err
			}
			bb := new(bytes.Buffer)
			err = cfg.Codec.NewEncoder(bb).Encode(&dto{
				Request:  req,
				Response: v,
			})
			if err != nil {
				return nil, 0, err
			}
			return bb.Bytes(), ttl, nil
		}))
		if err != nil {
			return zero, false, err
		}
		curr, err := hash(req)
		if err != nil {
			return zero, false, err
		}
		var res dto
		if err := cfg.Codec.NewDecoder(bytes.NewReader(b)).Decode(&res); err != nil {
			return zero, false, err
		}
		prev, err := hash(res.Request)
		if err != nil {
			return zero, false, err
		}
		if curr != prev {
			return zero, false, fmt.Errorf("%w: request", ErrConflict)
		}

		return res.Response, loaded, nil
	}, nil
}

// GoFunc memoizes fn in any byte [Storage] (typically [FS] or [File]) without
// expiry. The key is derived from fn's name and the JSON form of the
// argument, so calls with equal arguments share one result, also across
// program runs when the storage is persistent. A nil codec defaults to
// [JSONCodec]. Errors are not cached.
//
// Because the key uses the function's name, two different closures created
// at the same source location share keys; wrap only named functions or
// closures that capture no varying state.
func GoFunc[K, V any](fn func(context.Context, K) (V, error), c Storage[[]byte], codec Codec) func(context.Context, K) (V, error) {
	if codec == nil {
		codec = NewJSONCodec()
	}
	return func(ctx context.Context, args K) (V, error) {
		var zero V
		b, err := orderedJSON(args)
		if err != nil {
			return zero, err
		}
		key := helper.DigestString(fmt.Sprintf("%s:%s", getFunctionName(fn), b))
		file := fmt.Sprintf("%d.dat", key)
		curr, _, err := c.LoadOrCreate(ctx, file, func(ctx context.Context, key string) ([]byte, time.Duration, error) {
			v, err := fn(ctx, args)
			if err != nil {
				return nil, 0, err
			}
			bb := new(bytes.Buffer)
			err = codec.NewEncoder(bb).Encode(v)
			if err != nil {
				return nil, 0, err
			}
			return bb.Bytes(), 0, nil
		})
		if err != nil {
			return zero, err
		}
		var v V
		err = codec.NewDecoder(bytes.NewBuffer(curr)).Decode(&v)
		if err != nil {
			return zero, err
		}

		return v, nil
	}
}

func getFunctionName(fn any) string {
	// Extract the PC pointer and look up its metadata
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func orderedJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var a any
	err = json.Unmarshal(b, &a)
	if err != nil {
		return nil, err
	}

	return json.Marshal(a, json.Deterministic(true))
}

func hash(v any) (string, error) {
	b, err := orderedJSON(v)
	if err != nil {
		return "", err
	}

	return fmt.Sprint(xxh3.Hash(b)), nil
}

// WithDefaults returns a copy of c with Codec defaulted to JSON.
func (c FuncConfig[K]) WithDefaults() FuncConfig[K] {
	if c.Codec == nil {
		c.Codec = NewJSONCodec()
	}
	return c
}

// Validate reports whether c has all required fields.
func (c FuncConfig[K]) Validate() error {
	if c.KeyFn == nil || c.Lock == nil || c.Codec == nil {
		return errors.New("cache: key function, lock, and codec are required")
	}
	_, err := leaseConfig(c.LeaseTTL, c.Wait, c.RefreshRatio)
	return err
}

// MustFunc is like [Func] but panics on error.
func MustFunc[K, V any](fn func(ctx context.Context, req K) (V, time.Duration, error), cfg FuncConfig[K]) func(ctx context.Context, req K) (V, bool, error) {
	f, err := Func(fn, cfg)
	if err != nil {
		panic(err)
	}
	return f
}

// MustIdempotent is like [Idempotent] but panics on error.
func MustIdempotent[K, V any](fn func(ctx context.Context, req K) (V, time.Duration, error), cfg FuncConfig[K]) func(ctx context.Context, req K) (V, bool, error) {
	f, err := Idempotent(fn, cfg)
	if err != nil {
		panic(err)
	}
	return f
}
