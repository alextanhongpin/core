// Package core is the documentation entry point for Go libraries for HTTP
// services, concurrency control, Redis coordination, and typed utilities.
//
// Each library belongs to an independently versioned module. Install the module
// you need, for example:
//
//	go get github.com/alextanhongpin/core/sync/throttle
//
// The root module provides this overview. It does not import or bundle the
// independently released libraries.
//
// # Modules
//
// HTTP helpers for authentication, middleware, requests, responses, and servers:
// https://pkg.go.dev/github.com/alextanhongpin/core/http
//
// A/B experiments, adaptive bandits, analytics, and recommendations:
// https://pkg.go.dev/github.com/alextanhongpin/core/ab
//
// In-process concurrency and coordination:
//
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/broadcast
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/cache
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/circuitbreaker
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/dataloader
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/flock
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/goroutine
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/lock
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/pipeline
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/promise
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/rate
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/ratelimit
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/retry
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/snapshot
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/throttle
//   - https://pkg.go.dev/github.com/alextanhongpin/core/sync/timer
//
// Redis-backed coordination and probabilistic structures:
//
//   - https://pkg.go.dev/github.com/alextanhongpin/core/dsync/cache
//   - https://pkg.go.dev/github.com/alextanhongpin/core/dsync/channel
//   - https://pkg.go.dev/github.com/alextanhongpin/core/dsync/circuitbreaker
//   - https://pkg.go.dev/github.com/alextanhongpin/core/dsync/idempotent
//   - https://pkg.go.dev/github.com/alextanhongpin/core/dsync/lock
//   - https://pkg.go.dev/github.com/alextanhongpin/core/dsync/probs
//   - https://pkg.go.dev/github.com/alextanhongpin/core/dsync/ratelimit
//   - https://pkg.go.dev/github.com/alextanhongpin/core/dsync/singleflight
//
// Typed values, collections, validation, and other utilities:
// https://pkg.go.dev/github.com/alextanhongpin/core/types
//
// Metrics and tracing:
//
//   - https://pkg.go.dev/github.com/alextanhongpin/core/metrics
//   - https://pkg.go.dev/github.com/alextanhongpin/core/telemetry
//
// YAML loading with includes and merges:
// https://pkg.go.dev/github.com/alextanhongpin/core/codec/yaml
//
// MHTML archive parsing:
// https://pkg.go.dev/github.com/alextanhongpin/core/text/mhtml
package core
