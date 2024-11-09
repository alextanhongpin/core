package ab

import (
	"github.com/spaolacci/murmur3"
)

func Hash(key string, size uint64) uint64 {
	return murmur3.Sum64([]byte(key)) % size
}

func Rollout(key string, percentage uint64) bool {
	return percentage > 0 && Hash(key, 100) <= percentage
}
