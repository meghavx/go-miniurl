package analytics

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"url-shortener/internal/events"
)

const ClickChannel = "click_events"

// PublishClickEvent sends a non-blocking event to Redis Pub/Sub
func PublishClickEvent(rdb *redis.Client, id uint64) {
	// Runs in the background
	go func() {
		event := events.ClickEvent{
			ID: id,
			TS: time.Now().UTC().Format(time.RFC3339),
		}
		payload, _ := json.Marshal(event)

		// Create context with timeout to ensure that goroutine never hangs
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Publish the event
		_ = rdb.Publish(ctx, "click_events", payload).Err()
		log.Println("Published Click event for id:", id)
	}()
}
