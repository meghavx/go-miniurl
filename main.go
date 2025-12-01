package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/mattn/go-sqlite3"
	"github.com/redis/go-redis/v9"
)

var chars = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func main() {
	// Setup SQLite db
	db, err := sql.Open("sqlite3", "urls.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			long_url TEXT NOT NULL
		);
	`); err != nil {
		log.Fatal("Failed to create table:", err)
	}

	// Setup Redis client
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	log.Println("Server started on localhost:8080")

	// Router
	router := chi.NewRouter()
	router.Use(middleware.Logger)

	// UI Related
	router.Handle("/static/*", http.StripPrefix(
		"/static/",
		http.FileServer(http.Dir("./static"))),
	)
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	// App routes
	router.Post("/shorten-url", func(w http.ResponseWriter, r *http.Request) {
		log.Println("====== POST/shorten-url Request ======")
		shortenUrlHandler(w, r, db, rdb, ctx)
	})

	router.Get("/{code}", func(w http.ResponseWriter, r *http.Request) {
		log.Println("====== GET/{code} Request ======")
		redirectUrlHandler(w, r, db, rdb, ctx)
	})

	// Serve the app
	if err = http.ListenAndServe(":8080", router); err != nil {
		log.Fatal("Failed to start server", err)
		return
	}
}

// Handlers
func shortenUrlHandler(
	w http.ResponseWriter,
	r *http.Request,
	db *sql.DB,
	rdb *redis.Client,
	ctx context.Context,
) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	longUrl := strings.TrimSpace(r.FormValue("url"))
	if longUrl == "" {
		http.Error(w, "URL required", http.StatusBadRequest)
		return
	}

	longKey := "long:" + longUrl

	// Try Redis: longUrl → id
	log.Println("Trying Redis")

	if cachedID, err := rdb.Get(ctx, longKey).Result(); err == nil {
		log.Println("Redis hit // Short URL found in cache")
		id, _ := strconv.ParseInt(cachedID, 10, 64)
		writeShortUrl(w, r, id)
		return
	}

	// Redis miss -> Try SQLite
	log.Println("Redis miss // Trying SQLite")

	var id int64
	if err := db.QueryRow("SELECT id FROM urls WHERE long_url = ?", longUrl).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Println("SQLite miss // New URL --> Shorten + Persist to DB + Cache ")
			res, err := db.Exec("INSERT INTO urls(long_url) VALUES(?)", longUrl)
			if err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
			id, _ = res.LastInsertId()
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}
	code := base62Encode(uint64(id))
	shortKey := "short:" + code

	// Cache both mappings to Redis
	_ = rdb.Set(ctx, longKey, fmt.Sprintf("%d", id), 0).Err()
	_ = rdb.Set(ctx, shortKey, longUrl, 0).Err()

	writeShortUrl(w, r, id)
}

func redirectUrlHandler(
	w http.ResponseWriter,
	r *http.Request,
	db *sql.DB,
	rdb *redis.Client,
	ctx context.Context,
) {
	code := chi.URLParam(r, "code")

	// Try Redis
	log.Println("Trying Redis")

	if longUrl, err := rdb.Get(ctx, "short:"+code).Result(); err == nil {
		log.Println("Redis hit // Long URL found in cache")
		http.Redirect(w, r, longUrl, http.StatusMovedPermanently)
		return
	}

	// Redis miss -> Try SQLite
	log.Println("Redis miss // Trying SQLite")

	var longUrl string
	id := base62Decode(code)
	if err := db.QueryRow("SELECT long_url FROM urls WHERE id = ?", id).Scan(&longUrl); err != nil {
		http.NotFound(w, r)
		return
	}

	// Cache short code to Redis
	_ = rdb.Set(ctx, "short:"+code, longUrl, 0).Err()

	// Redirect to original url
	http.Redirect(w, r, longUrl, http.StatusMovedPermanently)
}

// Helper functions
func writeShortUrl(w http.ResponseWriter, r *http.Request, id int64) {
	code := base62Encode(uint64(id))

	protocol := r.Header.Get("X-Forwarded-Proto")
	if r.TLS == nil {
		protocol = "http"
	}
	shortUrl := fmt.Sprintf("%s://%s/%s", protocol, r.Host, code)

	// Load and render two.html
	tmpl := template.Must(template.ParseFiles("static/pages/two.html"))
	w.WriteHeader(http.StatusCreated)
	tmpl.Execute(w, map[string]string{
		"ShortURL": shortUrl,
	})
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
