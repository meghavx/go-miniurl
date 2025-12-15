package http

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	//"url-shortener/internal/analytics"
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
			storeShortAndLongKeysInRedis(ctx, rdb, code, longURL, id)
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

	/* // Publish click event
	analytics.PublishClickEvent(rdb, id) */

	// Update DB with click count
	ts := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.ExecContext(ctx, `
		UPDATE urls
		SET click_count = COALESCE(click_count, 0) + 1, last_visited_at = ?
		WHERE id = ?`,
		ts, id); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
	}

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
	id := utils.Base62Decode(code)
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

func retrieveLongURL(ctx context.Context, db *sql.DB, rdb *redis.Client, code string) (uint64, string) {
	id := utils.Base62Decode(code)
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

	hashedURL := utils.HashURL(longURL)
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
	protocol := "https"
	if r.TLS == nil {
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
