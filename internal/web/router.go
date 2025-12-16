package web

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	"url-shortener/internal/middleware/ratelimit"
)

func New(db *sql.DB, rdb *redis.Client) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Logger)

	// static files
	router.Handle("/static/*",
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))),
	)

	// index
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	// ---------------- RATE LIMITED GROUP -----------------
	router.Group(func(sub chi.Router) {
		sub.Use(ratelimit.Global(rdb, 50, time.Minute))

		// shorten url
		sub.With(ratelimit.PerIP(rdb, 10, time.Minute)).
			Post("/shorten-url", func(w http.ResponseWriter, r *http.Request) {
				ShortenURL(w, r, db, rdb)
			})

		// track clicks
		sub.Post("/track-clicks", func(w http.ResponseWriter, r *http.Request) {
			TrackClicks(w, r, db)
		})

		// redirect
		sub.Get("/{code}", func(w http.ResponseWriter, r *http.Request) {
			code := chi.URLParam(r, "code")
			RedirectURL(w, r, code, db, rdb)
		})

		// preview
		sub.Post("/preview-url", func(w http.ResponseWriter, r *http.Request) {
			PreviewURL(w, r, db, rdb)
		})
	})

	return router
}
