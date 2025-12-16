package main

import (
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"

	"url-shortener/internal/bloom"
	"url-shortener/internal/db"
	router "url-shortener/internal/web"
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

	port := ":8080"

	log.Printf("Server started on localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
