package cache_test

import (
	"testing"
	"time"

	"github.com/alextanhongpin/core/storage/cache"
	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	assert := assert.New(t)

	key := cache.NewKey("users:%d", 1*time.Minute)
	assert.Equal("users:42", key.Format(42))
	assert.Equal(1*time.Minute, key.TTL)
}

func TestTypedKey(t *testing.T) {
	assert := assert.New(t)

	type User struct {
		ID int
	}

	key := cache.NewTypedKey[User]("users:{{.ID}}", 1*time.Minute)
	assert.Equal("users:42", key.Format(User{ID: 42}))
	assert.Equal(1*time.Minute, key.TTL)
}
