module github.com/alextanhongpin/core/dsync/lock/examples

go 1.24.2

require (
	github.com/alextanhongpin/core/dsync/lock v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.9.0
)

require (
	github.com/alextanhongpin/core/dsync/cache v0.0.0-20250530081951-9764c3eb58c7 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/google/uuid v1.6.0 // indirect
)

replace github.com/alextanhongpin/core/dsync/lock => ../
