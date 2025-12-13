package db

import (
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

func InitRedis() *redis.Client {
	if url := os.Getenv("REDIS_URL"); url != "" {
		opt, err := redis.ParseURL(url)
		if err != nil {
			log.Fatal("Invalid REDIS_URL", err)
		}
		return redis.NewClient(opt)
	}

	// fallback for local dev
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}
