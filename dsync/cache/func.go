package cache

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"time"
)

type handler[K, V any] = func(context.Context, K) (V, error)

func Func2[K, V any](fn handler[K, V], c cache[[]byte], codec Codec) handler[K, V] {
	return func(ctx context.Context, args K) (V, error) {
		var zero V
		b, err := jsonStringify(args)
		if err != nil {
			return zero, err
		}
		name := hash(fmt.Appendf(nil, "%s:%s", getFunctionName(fn), b))
		file := fmt.Sprintf("%s.dat", name)
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

type FuncConfig struct {
	CacheDir string
	Codec    Codec
}

func Func[K, V any](fn handler[K, V], cfg *FuncConfig) handler[K, V] {
	cfg = cmp.Or(cfg, new(FuncConfig))
	cacheDir := cmp.Or(cfg.CacheDir, ".cache")
	var codec Codec = NewGobCodec()
	if cfg.Codec != nil {
		codec = cfg.Codec
	}
	mkdirAll := sync.OnceValue(func() error {
		return os.MkdirAll(cacheDir, 0o755)
	})
	return func(ctx context.Context, args K) (V, error) {
		var zero V
		if err := mkdirAll(); err != nil {
			return zero, err
		}

		b, err := jsonStringify(args)
		if err != nil {
			return zero, err
		}
		name := hash(fmt.Appendf(nil, "%s:%s", getFunctionName(fn), b))
		file := filepath.Join(cacheDir, fmt.Sprintf("%s.dat", name))
		f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			f, err := os.Open(file)
			if err != nil {
				return zero, err
			}
			defer func() {
				_ = f.Close()
			}()
			var v V
			err = codec.NewDecoder(f).Decode(&v)
			if err != nil {
				return zero, err
			}
			return v, nil
		}
		defer func() {
			_ = f.Close()
		}()
		v, err := fn(ctx, args)
		if err != nil {
			return zero, err
		}
		err = codec.NewEncoder(f).Encode(v)
		if err != nil {
			return zero, err
		}

		return v, nil
	}
}

func jsonStringify(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var a any
	err = json.Unmarshal(b, &a)
	if err != nil {
		return nil, err
	}

	return json.Marshal(a)
}

func getFunctionName(fn any) string {
	// Extract the PC pointer and look up its metadata
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func hash(data []byte) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
