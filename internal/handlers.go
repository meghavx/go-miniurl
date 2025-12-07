package internal

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"url-shortener/internal/utils"
)

func ShortenURL(w http.ResponseWriter, r *http.Request, db *sql.DB, rdb *redis.Client) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	longURL := strings.TrimSpace(r.FormValue("url"))
	if longURL == "" {
		http.Error(w, "URL required", http.StatusBadRequest)
		return
	}

	if !utils.IsValidURL(longURL) {
		http.Error(w, "Invalid or unsafe URL", http.StatusBadRequest)
		return
	}

	var id int64
	longKey := "long_to_id:" + longURL

	// Try Redis
	if cachedID, err := rdb.Get(ctx, longKey).Result(); err == nil {
		id, _ = strconv.ParseInt(cachedID, 10, 64)
		code := utils.Base62Encode(uint64(id))
		writeShortURL(w, r, code)
		return
	}

	// Try SQLite
	if err := db.QueryRow("SELECT id FROM urls WHERE long_url = ?", longURL).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			res, err := db.Exec("INSERT INTO urls(long_url) VALUES(?)", longURL)
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

	// Encode id to a short code
	code := utils.Base62Encode(uint64(id))
	shortKey := "code_to_long:" + code

	// Set ttl to 24 hours
	ttl := 24 * time.Hour

	// Write both mappings to redis cache
	_ = rdb.Set(ctx, shortKey, longURL, ttl).Err()
	_ = rdb.Set(ctx, longKey, fmt.Sprint(id), ttl).Err()

	// Write response to html template
	writeShortURL(w, r, code)
}

func RedirectURL(w http.ResponseWriter, r *http.Request, code string, db *sql.DB, rdb *redis.Client) {
	ctx := r.Context()
	var longURL string
	key := "code_to_long:" + code

	// Try Redis
	if longURL, err := rdb.Get(ctx, key).Result(); err == nil {
		http.Redirect(w, r, longURL, http.StatusMovedPermanently)
		return
	}

	// Try SQLite
	id := utils.Base62Decode(code)
	if db.QueryRowContext(ctx, "SELECT long_url FROM urls WHERE id = ?", id).Scan(&longURL) != nil {
		http.NotFound(w, r)
		return
	}

	// Cache and redirect
	_ = rdb.Set(ctx, key, longURL, 0).Err()
	http.Redirect(w, r, longURL, http.StatusMovedPermanently)
}

func writeShortURL(w http.ResponseWriter, r *http.Request, code string) {
	protocol := "https"
	if r.TLS == nil {
		protocol = "http"
	}
	shortURL := fmt.Sprintf("%s://%s/%s", protocol, r.Host, code)
	tmpl := template.Must(template.ParseFiles("static/partials/shorten_result.html"))
	w.WriteHeader(http.StatusCreated)
	_ = tmpl.Execute(w, map[string]string{
		"ShortURL": shortURL,
	})
}
