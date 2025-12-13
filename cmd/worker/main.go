package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"url-shortener/internal/analytics"
	"url-shortener/internal/db"
	"url-shortener/internal/events"
)

func main() {
	ctx := context.Background()

	rdb := db.InitRedis()
	sqlite := db.InitSQLite()
	defer sqlite.Close()

	// Subscribe to click events
	sub := rdb.Subscribe(ctx, analytics.ClickChannel)
	defer sub.Close()

	channel := sub.Channel()

	// Graceful shutdown
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Worker: listening for click_events...")

runLoop:
	for {
		select {
		case msg, ok := <-channel:
			if !ok {
				break runLoop
			}
			go handleMessage(sqlite, msg.Payload)
		case s := <-sigChannel:
			fmt.Println("Worker: signal", s, "shutting down...")
			break runLoop
		}
	}
	time.Sleep(200 * time.Millisecond)
	fmt.Println("Worker stopped")
}

func handleMessage(dbConn *sql.DB, payload string) {
	var event events.ClickEvent

	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		fmt.Println("Worker: invalid event:", err)
		return
	}

	_, err := dbConn.Exec(`
		UPDATE urls
		SET click_count = COALESCE(click_count, 0) + 1, 
		    last_visited_at = ?
		WHERE id = ?`,
		event.TS, event.ID,
	)

	if err != nil {
		fmt.Println("Worker: db update error:", err)
	}
}
