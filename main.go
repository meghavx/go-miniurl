package main

import (
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"

	router "url-shortener/internal"
	"url-shortener/internal/bloom"
	"url-shortener/internal/db"
)

func main() {
	sqlite := db.InitSQLite()
	rdb := db.InitRedis()

	bloom.InitBloom(1_000_000, 0.01)
	if err := bloom.Populate(sqlite); err != nil {
		log.Println("Bloom populate failed: " + err.Error())
	}
	log.Println("Bloom enabled?", bloom.Enabled)

	r := router.New(sqlite, rdb)

	log.Println("Server started on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
