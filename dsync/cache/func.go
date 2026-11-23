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
)

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
