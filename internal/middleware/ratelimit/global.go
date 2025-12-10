package ratelimit

import (
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// Global returns a fixed-window rate limiting middleware
// It enforces a global request limit for all incoming HTTP traffic
func Global(rdb *redis.Client, limit int64, window time.Duration) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Key to store ONE counter for the entire app
			// Tracks no. of requests made in current window
			key := "ratelimit:global"

			// Incr global counter by 1, and get the new value
			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// Redis down -> Let the request through
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				// First request in current window -> Set TTL
				rdb.Expire(ctx, key, window)
			}

			// Total requests exceeded the limit -> Return 429
			if count > limit {
				ttl, _ := rdb.TTL(ctx, key).Result()
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Global rate limit exceeded. Please try again after " + ttl.String()))
				return
			}

			// Total requests under limit -> Let the request through
			next.ServeHTTP(w, r)
		})
	}
}
