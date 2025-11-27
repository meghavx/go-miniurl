package main

import (
	"log"
	"net/http" // used to create http clients and servers

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// initialize a new chi router
	// --methods starting with new are usually constructors
	router := chi.NewRouter()

	// To print request logs on terminal
	router.Use(middleware.Logger)

	// register a GET route for the root ("/") path
	router.Get("/", basicHandler)

	// http server instantiation
	server := &http.Server{
		// server properties
		Addr:    ":3000",
		Handler: router,
	}

	// run the server
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("Failed to start server", err)
	}
}

func basicHandler(w http.ResponseWriter, r *http.Request) {
	// Type conversion: string -> byte array
	// comprising the ascii codes of each char
	w.Write([]byte("Hello from Go!\n"))
}
