package cache

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/redis/go-redis/v9/helper"
	"github.com/zeebo/xxh3"
)

type fun[K, V any] = func(ctx context.Context, req K) (V, time.Duration, error)
type ifun[K, V any] = func(ctx context.Context, req K) (V, bool, error)
type gofun[K, V any] = func(context.Context, K) (V, error)
type keyfun[K any] = func(ctx context.Context, req K) (string, error)

type FuncConfig[K any] struct {
	Codec Codec
	KeyFn keyfun[K]
	Lock  *Lock
}

func Func[K, V any](fn fun[K, V], cfg *FuncConfig[K]) ifun[K, V] {
	return func(ctx context.Context, args K) (V, bool, error) {
		var zero V
		key, err := cfg.KeyFn(ctx, args)
		if err != nil {
			return zero, false, err
		}

		curr, loaded, err := cfg.Lock.LoadOrCreate(ctx, key, &LoadOrCreateConfig[[]byte]{
			Create: func(ctx context.Context, key string) ([]byte, time.Duration, error) {
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
			},
		})
		if err != nil {
			return zero, false, err
		}
		var v V
		err = cfg.Codec.NewDecoder(bytes.NewBuffer(curr)).Decode(&v)
		if err != nil {
			return zero, false, err
		}

		return v, loaded, nil
	}
}

func Idempotent[K, V any](fn fun[K, V], cfg *FuncConfig[K]) ifun[K, V] {
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
		b, loaded, err := cfg.Lock.LoadOrCreate(ctx, key, &LoadOrCreateConfig[[]byte]{
			Create: func(ctx context.Context, key string) ([]byte, time.Duration, error) {
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
			},
		})
		if err != nil {
			return zero, false, err
		}
		curr, err := hash(req)
		if err != nil {
			return zero, false, err
		}
		var res dto
		if err := json.Unmarshal(b, &res); err != nil {
			return zero, false, err
		}
		prev, err := hash(res.Request)
		if err != nil {
			return zero, false, err
		}
		if curr != prev {
			return zero, false, fmt.Errorf("%w: request", ErrConflict)
		}

		return res.Response, loaded, err
	}
}

func GoFunc[K, V any](fn gofun[K, V], c cache[[]byte], codec Codec) gofun[K, V] {
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
