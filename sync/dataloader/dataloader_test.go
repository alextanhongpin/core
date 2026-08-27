package dataloader_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/dataloader"
	"github.com/alextanhongpin/evaltest"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestDataLoader(t *testing.T) {
	batchFn := func(ctx context.Context, keys []string) (res map[string]int, err error) {
		slices.Sort(keys)

		defer func() {
			keys := slices.Sorted(maps.Keys(res))
			var output []any
			for _, key := range keys {
				output = append(output, key, res[key])
			}
			evaltest.Log(ctx, evaltest.NewT("batchFn", keys, output, err))
		}()

		if strings.Contains(evaltest.Name(ctx), "error") {
			return nil, errors.ErrUnsupported
		}

		m := make(map[string]int)
		for _, k := range keys {
			n, err := strconv.Atoi(k)
			if err != nil {
				continue
			}
			m[k] = n
		}

		return m, nil
	}
	evaltest.Run(t, func(t *testing.T, ctx context.Context, input []string) (any, error) {
		dl, stop := dataloader.New(ctx, batchFn, &dataloader.Config{
			BatchInterval: 16 * time.Millisecond,
			BatchSize:     5,
		})
		defer stop()
		name := evaltest.Name(ctx)

		if strings.Contains(name, "many") {
			return dl.LoadMany(input...)
		}

		res := make([]*dataloader.Result[string, int], len(input))
		var wg sync.WaitGroup
		for i, k := range input {
			wg.Go(func() {
				time.Sleep(time.Duration(i) * time.Millisecond)
				v, err := dl.Load(k)
				res[i] = &dataloader.Result[string, int]{Key: k, Value: v, Error: err}
			})
		}
		wg.Wait()
		return res, nil
	})
}

func TestDataloader_Func(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}

	var batches int
	batchFn := func(ctx context.Context, keys []string) (res map[string]User, err error) {
		batches++
		t.Log("Loading", keys)
		slices.Sort(keys)
		m := make(map[string]User)
		for i, k := range keys {
			m[k] = User{ID: i, Name: k}
		}

		return m, nil
	}

	fn := func(ctx context.Context, u User) (string, error) {
		return fmt.Sprintf("hi, %s", u.Name), nil
	}

	dl, stop := dataloader.New(ctx, batchFn, &dataloader.Config{
		BatchInterval: 16 * time.Millisecond,
		BatchSize:     5,
	})
	defer stop()

	f := dataloader.Func(fn, dl)

	var wg sync.WaitGroup
	for _, name := range []string{"alice", "bob", "john"} {
		wg.Go(func() {
			t.Log(f(ctx, name))
		})
	}
	for _, name := range []string{"alice", "bob", "charles"} {
		wg.Go(func() {
			t.Log(f(ctx, name))
		})
	}
	wg.Wait()

	runtime.GC()

	t.Log("after GC")
	for _, name := range []string{"alice", "bob", "john"} {
		wg.Go(func() {
			t.Log(f(ctx, name))
		})
	}
	wg.Wait()
	if want, got := batches, 2; want != got {
		t.Fatalf("want %d batches, got %d", want, got)
	}
}

func TestDataloader_ErrNotFound(t *testing.T) {
	is := assert.New(t)
	dl, stop := newDataloader(func(ctx context.Context, keys []string) (map[string]int, error) {
		return newBatchFn(ctx, nil)
	})
	defer stop()

	v, err := dl.Load("1")
	is.Empty(v)
	is.ErrorIs(err, dataloader.ErrNotFound)
}

func TestDataloader_ErrCanceled(t *testing.T) {
	is := assert.New(t)
	dl, stop := newDataloader(newBatchFn)
	stop()

	v, err := dl.Load("1")
	is.Empty(v)
	is.ErrorIs(err, dataloader.ErrCanceled)
}

func TestDataloader_Error(t *testing.T) {
	is := assert.New(t)
	dl, stop := newDataloader(newBatchFn)
	defer stop()

	v, err := dl.Load("abc")
	is.Empty(v)
	is.Error(err)
	t.Log(err)
}

func newDataloader(batchFn func(context.Context, []string) (map[string]int, error)) (*dataloader.DataLoader[string, int], func()) {
	return dataloader.New(ctx, batchFn, &dataloader.Config{
		BatchInterval: 16 * time.Millisecond,
		BatchSize:     5,
	})
}

func newBatchFn(ctx context.Context, keys []string) (map[string]int, error) {
	m := make(map[string]int)
	for _, k := range keys {
		n, err := strconv.Atoi(k)
		if err != nil {
			return nil, err
		}
		m[k] = n
	}

	return m, nil
}
