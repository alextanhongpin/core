package dataloader_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/alextanhongpin/core/sync/dataloader"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestDataloader(t *testing.T) {
	is := assert.New(t)
	dl := newDataloader(func(ctx context.Context, keys []string) (map[string]int, error) {
		is.ElementsMatch(keys, []string{"1", "2", "3", "4", "5"})
		is.Len(keys, 5)

		return newBatchFn(ctx, keys)
	})
	defer dl.Stop()

	n := 10

	var wg sync.WaitGroup
	wg.Add(n)

	for i := range n {
		go func() {
			defer wg.Done()

			id := (i % 5) + 1
			t.Log(id)
			key := strconv.Itoa(id)
			v, err := dl.Load(key)
			is.Nil(err)
			is.Equal(id, v)
		}()
	}

	wg.Wait()
}

func TestDataloader_BatchMaxKeys(t *testing.T) {
	is := assert.New(t)

	var i int
	test := func(keys []string) {
		defer func() {
			i++
		}()

		if i == 0 {
			is.ElementsMatch(keys, []string{"1", "2", "3", "4", "5"})
			is.Len(keys, 5)
			return
		}

		is.ElementsMatch(keys, []string{"6", "7", "8", "9", "10"})
		is.Len(keys, 5)
	}
	dl := newDataloader(func(ctx context.Context, keys []string) (map[string]int, error) {
		test(keys)

		return newBatchFn(ctx, keys)
	})
	defer dl.Stop()

	n := 10

	var wg sync.WaitGroup
	wg.Add(n)

	for i := range n {
		go func() {
			defer wg.Done()

			time.Sleep(time.Duration(i) * time.Millisecond)

			id := i + 1
			t.Log(id)
			key := strconv.Itoa(id)
			v, err := dl.Load(key)
			is.Nil(err)
			is.Equal(id, v)
		}()
	}

	wg.Wait()
}

func newDataloader(batchFn func(context.Context, []string) (map[string]int, error)) *dataloader.DataLoader[string, int] {
	return dataloader.New[string, int](ctx, &dataloader.Options[string, int]{
		BatchFn:      batchFn,
		BatchTimeout: 16 * time.Millisecond,
		BatchMaxKeys: 5,
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
