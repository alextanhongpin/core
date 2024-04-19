package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/alextanhongpin/core/sync/ratelimit"
)

func main() {
	rl := ratelimit.NewTokenBucket(5, time.Minute, 3)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		res := rl.Allow()

		reset := int(time.Now().Sub(res.ResetAt).Seconds())
		if reset < 0 {
			reset = 0
		}
		fmt.Printf("%+v\n", res)

		if !res.Allow {
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", res.Limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", reset))

			w.WriteHeader(http.StatusTooManyRequests)
		}

		now := time.Now()
		json.NewEncoder(w).Encode(map[string]any{
			"allow":     res.Allow,
			"t0":        now.Truncate(time.Minute),
			"ti":        now,
			"tn":        now.Truncate(time.Minute).Add(time.Minute),
			"percent":   float64(now.Sub(now.Truncate(time.Minute))) / float64(time.Minute),
			"limit":     res.Limit,
			"remaining": res.Remaining,
			"resetIn":   res.ResetAt.Sub(time.Now()).String(),
			"retryAt":   res.RetryAt.Sub(time.Now()).String(),
		})
	})

	fmt.Println("listening on :8080. press ctrl+c to cancel.")
	http.ListenAndServe(":8080", mux)
}
