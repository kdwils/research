package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
)

// Simple HTTP server to serve migration registry for testing

func main() {
	// Serve files from current directory
	fs := http.FileServer(http.Dir("."))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s %s", r.Method, r.URL.Path)
		fs.ServeHTTP(w, r)
	})

	addr := ":8080"
	fmt.Printf("Migration registry server starting on %s\n", addr)
	fmt.Printf("Example URL: http://localhost:8080/nginx-ingress/index.yaml\n")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
