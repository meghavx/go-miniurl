package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

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

	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

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

func greetHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/index.html")
}

func shortenUrlHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Read long URL from form-data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	longUrl := strings.TrimSpace(r.FormValue("url"))

	var id int64

	err = db.QueryRow("SELECT id FROM urls WHERE long_url = ?", longUrl).Scan(&id)

	if errors.Is(err, sql.ErrNoRows) {
		res, err := db.Exec("INSERT INTO urls(long_url) VALUES(?)", longUrl)
		if err != nil {
			log.Println("Failed to insert url into db:", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		id, _ = res.LastInsertId()
	}

	code := base62Encode(uint64(id))
	shortUrl := fmt.Sprintf("http://localhost:8080/%s", code)

	htmlSnippet := fmt.Sprintf(`
		<div class="p-4 bg-green-100 text-green-800 rounded">
			Short URL: 
			<a href="%s" target="_blank" class="underline font-semibold">%s</a>
		</div>
	`, shortUrl, shortUrl)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(htmlSnippet))
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
