package main

import (
	"fmt"
	"net/http"
	"os"
)

// Simple backend server that prints "hello from <host>" and listens on the PORT environment variable (default 8081)
func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8081"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("hello from %s\n", r.Host)
	})

	fmt.Printf("listening on %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
