package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitSQLite(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			long_url TEXT NOT NULL UNIQUE,
			click_count INTEGER DEFAULT 0,
			last_visited_at DATETIME
		);
	`)

	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	return db
}
