package web

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

	"url-shortener/internal/core"

	"github.com/redis/go-redis/v9"
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

	// Try Redis first
	longKey := "long_to_id:" + core.HashURL(longURL)
	if cachedID, err := rdb.Get(ctx, longKey).Result(); err == nil {
		id, err = strconv.ParseInt(cachedID, 10, 64)
		if err == nil {
			code := core.Base62Encode(uint64(id))
			writeShortURL(w, r, code)
			return
		}
	}

	// Redis miss/failure -> check bloom filter before trying SQLite
	if !core.BloomEnabled || core.MightExistInBloom(longURL) {
		// Try SQLite
		err := db.QueryRowContext(ctx, "SELECT id FROM urls WHERE long_url = ?", longURL).Scan(&id)
		if err == nil {
			code := core.Base62Encode(uint64(id))
			storeShortAndLongKeysInRedis(ctx, rdb, code, longURL, id)
			writeShortURL(w, r, code)
			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	// URL not found in SQLite, or Bloom confirmed it is absent -> Insert in DB
	res, err := db.ExecContext(ctx, "INSERT INTO urls(long_url) VALUES(?)", longURL)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	id, err = res.LastInsertId()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	code := core.Base62Encode(uint64(id))

	// Update Bloom and store in Redis
	core.AddToBloom(longURL)
	storeShortAndLongKeysInRedis(ctx, rdb, code, longURL, id)
	writeShortURL(w, r, code)
}

func RedirectURL(w http.ResponseWriter, r *http.Request, code string, db *sql.DB, rdb *redis.Client) {
	ctx := r.Context()

	id, longURL := retrieveLongURL(ctx, db, rdb, code)
	if longURL == "" {
		http.Error(w, "Link not found!", http.StatusNotFound)
		return
	}

	// Publish click event
	core.PublishClickEvent(rdb, id)

	http.Redirect(w, r, longURL, http.StatusFound)
}

func PreviewURL(w http.ResponseWriter, r *http.Request, db *sql.DB, rdb *redis.Client) {
	ctx := r.Context()

	// Validate request and get short code
	code, err := validatePreviewRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, longURL := retrieveLongURL(ctx, db, rdb, code)
	if longURL == "" {
		http.Error(w, "Link not found!", http.StatusNotFound)
		return
	}

	// Write Response
	htmlSnippet := fmt.Sprintf(`
		<div class="p-4 bg-green-100 text-green-700 rounded">
			<p class="mb-1 font-bold">Original URL:</p>
			<a href="%s" target="_blank" class="underline font-medium">%s</a>
		</div>
	`, longURL, longURL)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlSnippet))
}

func TrackClicks(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	ctx := r.Context()

	// Validate request and get short code
	code, err := validatePreviewRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := core.Base62Decode(code)
	totalClicks, lastVisited := retrieveClickStats(w, ctx, db, id)

	// Write Response
	htmlSnippet := fmt.Sprintf(`
		<div class="space-y-2 p-4 px-4 sm:px-6 md:px-8 bg-green-100 text-green-700 rounded">
			<div class="flex items-center gap-1">
				<span class="flex items-center gap-2 text-gray-600 font-semibold w-32 shrink-0">
					<i data-lucide="bar-chart-2" class="w-4 h-4"></i>Total Clicks
				</span>
				<span class="font-semibold">%d</span>
			</div>
			<div class="flex items-center gap-1">
				<span class="flex items-center gap-2 text-gray-600 font-semibold w-32 shrink-0">
					<i data-lucide="clock" class="w-4 h-4"></i>Last Visited
				</span>
				<span id="last-visited" data-utc="%s" class="font-semibold">—</span>
			</div>
		</div>
	`, totalClicks, lastVisited)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlSnippet))
}

/**** Helper Methods below ****/

func validateShortenRequest(r *http.Request) (string, error) {
	// Parse URL
	url, err := ParseAndGetURL(r)
	if err != nil {
		return "", err
	}
	// Validate URL
	url, err = core.ValidateLongURL(url)
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
	code, err := core.ValidateShortURL(url, r.Host)
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

func retrieveLongURL(ctx context.Context, db *sql.DB, rdb *redis.Client, code string) (uint64, string) {
	id := core.Base62Decode(code)
	var longURL string

	// Try Redis
	key := "code_to_long:" + code
	if longURL, err := rdb.Get(ctx, key).Result(); err == nil {
		return id, longURL
	}
	// Redis miss -> Try SQLite
	if db.QueryRowContext(ctx, "SELECT long_url FROM urls WHERE id = ?", id).Scan(&longURL) != nil {
		return 0, ""
	}
	storeShortKeyInRedis(ctx, rdb, code, longURL)
	return id, longURL
}

func retrieveClickStats(w http.ResponseWriter, ctx context.Context, db *sql.DB, id uint64) (int, string) {
	var (
		clickCount    int
		lastVisitedAt sql.NullTime
	)
	err := db.QueryRowContext(ctx, "SELECT click_count, last_visited_at FROM urls WHERE id = ?", id).
		Scan(&clickCount, &lastVisitedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Link not found!", http.StatusNotFound)
			return 0, ""
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return 0, ""
		}
	}
	lastVisited := "Never" // fallback value
	if lastVisitedAt.Valid {
		lastVisited = lastVisitedAt.Time.UTC().Format(time.RFC3339)
	}
	return clickCount, lastVisited
}

func storeShortAndLongKeysInRedis(ctx context.Context, rdb *redis.Client, code string, longURL string, id int64) {
	// Store code -> longURL mapping
	storeShortKeyInRedis(ctx, rdb, code, longURL)

	hashedURL := core.HashURL(longURL)
	longKey := "long_to_id:" + hashedURL
	ttl := 24 * time.Hour

	// Store longURL -> id mapping
	_ = rdb.Set(ctx, longKey, fmt.Sprint(id), ttl).Err()
}

func storeShortKeyInRedis(ctx context.Context, rdb *redis.Client, code string, longURL string) {
	ttl := 24 * time.Hour
	shortKey := "code_to_long:" + code
	_ = rdb.Set(ctx, shortKey, longURL, ttl).Err()
}

func writeShortURL(w http.ResponseWriter, r *http.Request, code string) {
	protocol := r.Header.Get("X-Forwarded-Proto")
	if protocol == "" {
		protocol = "http"
	}
	shortURL := fmt.Sprintf("%s://%s/%s", protocol, r.Host, code)

	// Write Response
	htmlSnippet := fmt.Sprintf(`
		<div class="p-4 bg-green-100 text-green-700 rounded">
			<p class="mb-1 font-semibold">Short URL:</p> 
			<a href="%s" target="_blank" class="underline font-medium">%s</a>
		</div>
	`, shortURL, shortURL)

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(htmlSnippet))
}

var formTmpl = template.Must(
	template.ParseFiles("static/partials/url-form.html"),
)

func RenderForm(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Label       string
		Placeholder string
		Endpoint    string
	}{
		Label:       r.URL.Query().Get("label"),
		Placeholder: r.URL.Query().Get("placeholder"),
		Endpoint:    r.URL.Query().Get("endpoint"),
	}

	// Sensible defaults
	if data.Label == "" {
		data.Label = "Enter URL"
	}
	if data.Placeholder == "" {
		data.Placeholder = "https://example.com"
	}
	if data.Endpoint == "" {
		data.Endpoint = "/shorten-url"
	}

	w.Header().Set("Content-Type", "text/html")
	_ = formTmpl.Execute(w, data)
}
