module realworld

go 1.24.2

require github.com/alextanhongpin/core/dsync/cache v0.0.0

replace github.com/alextanhongpin/core/dsync/cache => ../../

require github.com/redis/go-redis/v9 v9.9.0

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)
