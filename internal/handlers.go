package internal

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"

	"url-shortener/internal/utils"
)

func ShortenURL(w http.ResponseWriter, r *http.Request, db *sql.DB, rdb *redis.Client) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	longUrl := strings.TrimSpace(r.FormValue("url"))
	if longUrl == "" {
		http.Error(w, "URL required", http.StatusBadRequest)
		return
	}

	var id int64
	longKey := "long:" + longUrl

	// Try Redis
	if cachedID, err := rdb.Get(ctx, longKey).Result(); err == nil {
		id, _ = strconv.ParseInt(cachedID, 10, 64)
		code := utils.Base62Encode(uint64(id))
		writeShortURL(w, r, code)
		return
	}

	// Try SQLite
	res, err := db.ExecContext(ctx, `INSERT INTO urls (long_url) VALUES (?) ON CONFLICT DO NOTHING`, longUrl)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	id, _ = res.LastInsertId()
	// id <= 0 ~ Insertion failed ~ URL exists in db -> Fetch that id
	if id <= 0 {
		err = db.QueryRowContext(ctx, "SELECT id FROM urls WHERE long_url = ?", longUrl).Scan(&id)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}
	// Encode id to a short code
	code := utils.Base62Encode(uint64(id))

	// Write both mappings to redis cache
	_ = rdb.Set(ctx, longKey, fmt.Sprint(id), 0).Err()
	_ = rdb.Set(ctx, "short:"+code, longUrl, 0).Err()

	// Write response to html template
	writeShortURL(w, r, code)
}

func RedirectURL(w http.ResponseWriter, r *http.Request, code string, db *sql.DB, rdb *redis.Client) {
	ctx := r.Context() // <<–– request-scoped context

	// Try Redis
	if longUrl, err := rdb.Get(ctx, "short:"+code).Result(); err == nil {
		http.Redirect(w, r, longUrl, http.StatusMovedPermanently)
		return
	}

	// Try SQLite
	var longUrl string
	id := utils.Base62Decode(code)

	err := db.QueryRowContext(ctx, "SELECT long_url FROM urls WHERE id = ?", id).Scan(&longUrl)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Cache
	_ = rdb.Set(ctx, "short:"+code, longUrl, 0).Err()

	http.Redirect(w, r, longUrl, http.StatusMovedPermanently)
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
