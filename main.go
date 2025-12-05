package main

import (
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"

	router "url-shortener/internal"
	"url-shortener/internal/db"
)

func main() {
	sqlite := db.InitSQLite()
	rdb := db.InitRedis()

	r := router.New(sqlite, rdb)

	log.Println("Server started on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
