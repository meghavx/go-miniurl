package internal

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
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

	// shorten url
	router.Post("/shorten-url", func(w http.ResponseWriter, r *http.Request) {
		ShortenURL(w, r, db, rdb)
	})

	// redirect
	router.Get("/{code}", func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		RedirectURL(w, r, code, db, rdb)
	})

	return router
}
