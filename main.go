package main

import (
	"log"
	"net/http" // used to create http clients and servers
)

func main() {

	// http server instantiation
	server := &http.Server{
		// server properties
		Addr:    ":3000",
		Handler: http.HandlerFunc(basicHandler),
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
