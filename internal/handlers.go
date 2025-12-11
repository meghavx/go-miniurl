package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"url-shortener/internal/bloom"
	"url-shortener/internal/utils"
)

func ShortenURL(w http.ResponseWriter, r *http.Request, db *sql.DB, rdb *redis.Client) {
	ctx := r.Context()
	var id int64

	// Validate request and get long URL
	longURL, err := validateShortenRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if bloom.MightExist(longURL) {
		// Try Redis
		longKey := "long_to_id:" + utils.HashURL(longURL)
		if cachedID, err := rdb.Get(ctx, longKey).Result(); err == nil {
			id, _ = strconv.ParseInt(cachedID, 10, 64)
			code := utils.Base62Encode(uint64(id))
			writeShortURL(w, r, code)
			return
		}

		// Redis miss -> Try SQLite
		err := db.QueryRow("SELECT id FROM urls WHERE long_url = ?", longURL).Scan(&id)
		if err == nil {
			code := utils.Base62Encode(uint64(id))
			storeShortAndLongKeysInRedis(rdb, ctx, code, longURL, id)
			writeShortURL(w, r, code)
			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	// Definitely a NEW URL -> Insert in DB
	res, err := db.Exec("INSERT INTO urls(long_url) VALUES(?)", longURL)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	id, _ = res.LastInsertId()
	code := utils.Base62Encode(uint64(id))

	// Store in Bloom and Redis
	bloom.Add(longURL)
	storeShortAndLongKeysInRedis(rdb, ctx, code, longURL, id)
	writeShortURL(w, r, code)
}

func RedirectURL(w http.ResponseWriter, r *http.Request, code string, db *sql.DB, rdb *redis.Client) {
	if longURL := retrieveLongURL(w, r, db, rdb, code); longURL != "" {
		ctx := r.Context()
		storeShortKeyInRedis(rdb, ctx, code, longURL)
		http.Redirect(w, r, longURL, http.StatusMovedPermanently)
	}
}

func PreviewURL(w http.ResponseWriter, r *http.Request, db *sql.DB, rdb *redis.Client) {
	// Validate request and get short code
	code, err := validatePreviewRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if longURL := retrieveLongURL(w, r, db, rdb, code); longURL != "" {
		ctx := r.Context()
		storeShortKeyInRedis(rdb, ctx, code, longURL)
		writeURLToTemplate(w, "LongURL", longURL, "preview_result.html")
	}
}

/**** Helper Methods below ****/

func validateShortenRequest(r *http.Request) (string, error) {
	// Parse URL
	url, err := ParseAndGetURL(r)
	if err != nil {
		return "", err
	}
	// Validate URL
	url, err = utils.ValidateLongURL(url)
	if err != nil {
		return url, err
	}
	return url, nil
}

func validatePreviewRequest(r *http.Request) (string, error) {
	// Parse URL
	url, err := ParseAndGetURL(r)
	if err != nil {
		return "", err
	}
	// Validate URL
	code, err := utils.ValidateShortURL(url, r.Host)
	if err != nil {
		return code, err
	}
	return code, nil
}

func ParseAndGetURL(r *http.Request) (string, error) {
	if err := r.ParseForm(); err != nil {
		return "", errors.New("Bad Request")
	}
	url := strings.TrimSpace(r.FormValue("url"))
	if url == "" {
		return "", errors.New("URL required")
	}
	return url, nil
}

func retrieveLongURL(w http.ResponseWriter, r *http.Request, db *sql.DB, rdb *redis.Client, code string) string {
	ctx := r.Context()
	var longURL string

	// Try Redis
	key := "code_to_long:" + code
	if longURL, err := rdb.Get(ctx, key).Result(); err == nil {
		return longURL
	}

	// Redis miss -> Try SQLite
	id := utils.Base62Decode(code)
	if db.QueryRowContext(ctx, "SELECT long_url FROM urls WHERE id = ?", id).Scan(&longURL) != nil {
		http.NotFound(w, r)
		return ""
	}
	return longURL
}

func storeShortAndLongKeysInRedis(rdb *redis.Client, ctx context.Context, code string, longURL string, id int64) {
	// Store code -> longURL mapping
	storeShortKeyInRedis(rdb, ctx, code, longURL)

	hashedURL := utils.HashURL(longURL)
	longKey := "long_to_id:" + hashedURL
	ttl := 24 * time.Hour

	// Store longURL -> id mapping
	_ = rdb.Set(ctx, longKey, fmt.Sprint(id), ttl).Err()
}

func storeShortKeyInRedis(rdb *redis.Client, ctx context.Context, code string, longURL string) {
	ttl := 24 * time.Hour
	shortKey := "code_to_long:" + code
	_ = rdb.Set(ctx, shortKey, longURL, ttl).Err()
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
