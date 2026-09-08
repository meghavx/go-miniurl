package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"url-shortener/internal/core"
	"url-shortener/internal/db"
	router "url-shortener/internal/web"
)

func main() {
	sqlitePath := os.Getenv("SQLITE_PATH")
	if sqlitePath == "" {
		sqlitePath = "./urls.db"
	}

	sqlite := db.InitSQLite(sqlitePath)
	rdb := db.InitRedis()

	core.InitBloom(1_000_000, 0.01)
	if err := core.PopulateBloom(sqlite); err != nil {
		log.Println("Bloom populate failed: " + err.Error())
	}
	log.Println("Bloom enabled?", core.BloomEnabled)

	r := router.New(sqlite, rdb)

	port := ":8080"

	log.Printf("Server started on localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
