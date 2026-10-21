// Komikan coming-soon page: a tiny HTTP server with an embedded HTML page.
//
// The page is rendered at request time from the embedded template and the
// board data, so the board shown on the page is always the real board.
package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/taolicus/komikan/internal/board"
)

//go:embed index.html favicon.svg
var content embed.FS

// boardMarker is replaced in the template by the rendered board.
const boardMarker = "<!--BOARD-->"

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:18082"
	}

	page, err := renderPage()
	if err != nil {
		log.Fatalf("komikan: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write(page); err != nil {
			log.Printf("komikan: %v", err)
		}
	})
	mux.Handle("/favicon.svg", http.FileServer(http.FS(content)))

	log.Printf("komikan: listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// renderPage fills the board marker in the template with the board rendered
// in its starting position.
func renderPage() ([]byte, error) {
	tpl, err := content.ReadFile("index.html")
	if err != nil {
		return nil, err
	}

	b, pos, err := board.Standard()
	if err != nil {
		return nil, err
	}
	if err := board.Validate(b, pos); err != nil {
		return nil, err
	}

	page := strings.Replace(string(tpl), boardMarker, b.RenderSVG(pos), 1)
	return []byte(page), nil
}
