package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitSQLite() *sql.DB {
	db, err := sql.Open("sqlite3", "urls.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			long_url TEXT NOT NULL UNIQUE
		);
	`)

	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	return db
}
