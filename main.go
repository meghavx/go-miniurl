package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/mattn/go-sqlite3"
)

var chars = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func main() {
	db, err := sql.Open("sqlite3", "urls.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTable := `
		CREATE TABLE IF NOT EXISTS urls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			long_url TEXT NOT NULL
		);
	`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Get("/", greetHandler)

	router.Post("/shorten-url", func(w http.ResponseWriter, r *http.Request) {
		shortenUrlHandler(w, r, db)
	})

	router.Get("/{code}", func(w http.ResponseWriter, r *http.Request) {
		redirectUrlHandler(w, r, db)
	})
	err = http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatal("Failed to start server", err)
	}
}

func greetHandler(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("Hello from Go!\n"))
}

func shortenUrlHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		Url string `json:"url"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	res, err := db.Exec("INSERT INTO urls(long_url) VALUES(?)", req.Url)
	if err != nil {
		log.Println("Failed to insert url into db:", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	id, _ := res.LastInsertId()

	code := base62Encode(uint64(id))
	shortUrl := fmt.Sprintf("http://localhost:3000/%s", code)

	resp := map[string]string{"shortUrl": shortUrl}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func redirectUrlHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	code := chi.URLParam(r, "code")
	id := base62Decode(code)
	var longUrl string
	err := db.QueryRow("SELECT long_url FROM urls WHERE id = ?", id).Scan(&longUrl)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, longUrl, http.StatusMovedPermanently)
}

func base62Encode(num uint64) string {
	if num == 0 {
		return "0"
	}
	res := make([]byte, 0)
	for num > 0 {
		res = append(res, chars[num%62])
		num /= 62
	}
	for i, j := 0, len(res)-1; i < j; i, j = i+1, j-1 {
		res[i], res[j] = res[j], res[i]
	}
	return string(res)
}

func base62Decode(s string) uint64 {
	var num uint64
	for i := 0; i < len(s); i++ {
		num = num*62 + uint64(bytes.IndexByte(chars, s[i]))
	}
	return num
}
