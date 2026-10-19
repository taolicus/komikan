// Komikan coming-soon page: a tiny HTTP server with an embedded HTML page.
package main

import (
	"embed"
	"log"
	"net/http"
	"os"
)

//go:embed index.html
var content embed.FS

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:18082"
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(content)))

	log.Printf("komikan: listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
