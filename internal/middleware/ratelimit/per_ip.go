package ratelimit

import (
	"net/http"
	"strconv"
	"time"
	"url-shortener/internal/utils"

	"github.com/redis/go-redis/v9"
)

// PerIP returns a sliding-window rate limiting middleware
// It enforces a ip level request limit for all incoming HTTP traffic
func PerIP(rdb *redis.Client, limit int64, window time.Duration) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get client IP
			ip := utils.GetIP(r)

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
