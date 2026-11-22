package cache_test

import (
	"fmt"

	"github.com/alextanhongpin/core/sync/cache"
)

func ExampleCache_LoadOrCreate() {
	c := cache.New(func(key string) (*string, error) {
		v := "value for " + key
		return &v, nil
	})

	v, loaded, err := c.LoadOrCreate("a")
	fmt.Println(*v, loaded, err)

	// While v is referenced the entry stays live, so the same pointer returns.
	v2, loaded, _ := c.LoadOrCreate("a")
	fmt.Println(v == v2, loaded)
	// Output:
	// value for a false <nil>
	// true true
}
