package internal

import (
	"context"
	"database/sql"
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
	var id int64

	longURL, provided := ParseAndGetURL(w, r)
	if !provided {
		return
	}
	if !utils.IsValidURL(longURL) {
		http.Error(w, "Invalid or unsafe URL", http.StatusBadRequest)
		return
	}

	// Try Redis
	longKey := "long_to_id:" + longURL
	if cachedID, err := rdb.Get(ctx, longKey).Result(); err == nil {
		id, _ = strconv.ParseInt(cachedID, 10, 64)
		code := utils.Base62Encode(uint64(id))
		writeShortURL(w, r, code)
		return
	}

	// Redis miss -> Try SQLite
	if err := db.QueryRow("SELECT id FROM urls WHERE long_url = ?", longURL).Scan(&id); err == nil && id > 0 {
		code := utils.Base62Encode(uint64(id))
		writeShortURL(w, r, code)
		return
	}

	// SQLite miss ~ New URL -> Insert in DB
	if res, err := db.Exec("INSERT INTO urls(long_url) VALUES(?)", longURL); err == nil {
		id, _ = res.LastInsertId()
		code := utils.Base62Encode(uint64(id))
		storeInRedis(rdb, ctx, code, longURL, &id)
		writeShortURL(w, r, code)
	} else {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
}

func RedirectURL(w http.ResponseWriter, r *http.Request, code string, db *sql.DB, rdb *redis.Client) {
	if longURL, found := retrieveLongURL(w, r, db, rdb, code); found {
		ctx := r.Context()
		storeInRedis(rdb, ctx, code, longURL, nil)
		http.Redirect(w, r, longURL, http.StatusMovedPermanently)
	}
}

func PreviewURL(w http.ResponseWriter, r *http.Request, db *sql.DB, rdb *redis.Client) {
	shortURL, provided := ParseAndGetURL(w, r)
	if !provided {
		return
	}
	code, err := utils.ExtractShortCode(shortURL)
	if err != nil {
		http.Error(w, "Invalid short URL", http.StatusBadRequest)
		return
	}
	if longURL, found := retrieveLongURL(w, r, db, rdb, code); found {
		writeURLToTemplate(w, "LongURL", longURL, "preview_result.html")
	}
}

/**** Helper Methods below ****/

func ParseAndGetURL(w http.ResponseWriter, r *http.Request) (string, bool) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return "", false
	}
	url := strings.TrimSpace(r.FormValue("url"))
	if url == "" {
		http.Error(w, "URL required", http.StatusBadRequest)
		return "", false
	}
	return url, true
}

func retrieveLongURL(w http.ResponseWriter, r *http.Request, db *sql.DB, rdb *redis.Client, code string) (string, bool) {
	ctx := r.Context()
	var longURL string

	// Try Redis
	key := "code_to_long:" + code
	if longURL, err := rdb.Get(ctx, key).Result(); err == nil {
		return longURL, true
	}

	// Redis miss -> Try SQLite
	id := utils.Base62Decode(code)
	if db.QueryRowContext(ctx, "SELECT long_url FROM urls WHERE id = ?", id).Scan(&longURL) != nil {
		http.NotFound(w, r)
		return "", false
	}
	return longURL, true
}

func storeInRedis(rdb *redis.Client, ctx context.Context, code string, longURL string, id *int64) {
	ttl := 24 * time.Hour

	// Store code -> longURL mapping
	shortKey := "code_to_long:" + code
	_ = rdb.Set(ctx, shortKey, longURL, ttl).Err()

	// Store longURL -> id mapping
	if id != nil {
		longKey := "long_to_id:" + longURL
		_ = rdb.Set(ctx, longKey, fmt.Sprint(id), ttl).Err()
	}
}

func writeShortURL(w http.ResponseWriter, r *http.Request, code string) {
	protocol := "https"
	if r.TLS == nil {
		protocol = "http"
	}
	shortURL := fmt.Sprintf("%s://%s/%s", protocol, r.Host, code)
	writeURLToTemplate(w, "ShortURL", shortURL, "shorten_result.html")
}

func writeURLToTemplate(w http.ResponseWriter, key string, val string, tmplName string) {
	tmpl := template.Must(template.ParseFiles("static/partials/" + tmplName))
	w.WriteHeader(http.StatusCreated)
	_ = tmpl.Execute(w, map[string]string{
		key: val,
	})
}
