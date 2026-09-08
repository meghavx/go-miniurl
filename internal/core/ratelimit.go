package core

import (
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Global returns a fixed-window rate limiting middleware
// It enforces a global request limit for all incoming HTTP traffic
func GlobalRateLimit(rdb *redis.Client, limit int64, window time.Duration) func(http.Handler) http.Handler {

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

// PerIP returns a sliding-window rate limiting middleware
// It enforces a ip level request limit for all incoming HTTP traffic
func PerIPRateLimit(rdb *redis.Client, limit int64, window time.Duration) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get client IP
			ip := GetIP(r)

			// Key for this IP's rate bucket
			key := "rate:ip" + ip

			now := float64(time.Now().UnixMilli())
			windowStart := now - float64(window.Milliseconds())
			startScore := strconv.FormatFloat(windowStart, 'f', -1, 64)

			// Remove timestamps older than current window
			rdb.ZRemRangeByScore(ctx, key, "0", startScore)

			endScore := strconv.FormatFloat(now, 'f', -1, 64)
			count, _ := rdb.ZCount(ctx, key, startScore, endScore).Result()

			// Total requests exceeded the limit -> Return 429
			if count > limit {
				ttl, _ := rdb.TTL(ctx, key).Result()
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Too many requests from your IP. Please try again after " + ttl.String()))
				return
			}

			// Add timestamp for current request
			rdb.ZAdd(ctx, key, redis.Z{Score: now, Member: now})

			// Set TTL with buffer so the key cannot expire mid-window
			rdb.Expire(ctx, key, window*2)

			next.ServeHTTP(w, r)
		})
	}
}
