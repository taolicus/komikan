// Package httpapi serves the HTTP interface of the Komikan application.
//
// The package owns routing and request handling. Game logic lives in
// internal/rules, persistence in internal/store, and the board data in
// internal/board; none of them know about HTTP.
package httpapi

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/taolicus/komikan/internal/board"
	"github.com/taolicus/komikan/web"
)

// boardMarker is replaced in the page template by the rendered board.
const boardMarker = "<!--BOARD-->"

// Config holds the dependencies of the server.
type Config struct {
	// Logger receives request and error logs. A nil logger discards them.
	Logger *slog.Logger
	// Version is the build version, reported by the version endpoint.
	Version string
}

// Server holds the state shared by the handlers.
type Server struct {
	log     *slog.Logger
	board   *board.Board
	page    []byte
	version string
	mux     *http.ServeMux
}

// New builds the server and its routes.
func New(cfg Config) (*Server, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	b, pos, err := board.Standard()
	if err != nil {
		return nil, fmt.Errorf("httpapi: load board: %w", err)
	}
	if err := board.Validate(b, pos); err != nil {
		return nil, fmt.Errorf("httpapi: invalid board: %w", err)
	}

	page, err := renderPage(b, pos)
	if err != nil {
		return nil, err
	}

	s := &Server{
		log:     logger,
		board:   b,
		page:    page,
		version: cfg.Version,
		mux:     http.NewServeMux(),
	}
	s.routes()
	return s, nil
}

// Handler returns the HTTP handler of the server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.handleHealthz)
	s.mux.HandleFunc("/", s.handleRoot)
	s.mux.Handle("/favicon.svg", http.FileServer(http.FS(web.Files)))
}

// handleRoot serves the coming-soon page. Other paths are left to the tasks
// that add them, such as the room and game pages.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if _, err := w.Write(s.page); err != nil {
		s.log.Error("write page", "error", err)
	}
}

// handleHealthz reports that the server is up.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "ok")
}

// renderPage fills the board marker in the page template with the board
// rendered in its starting position.
func renderPage(b *board.Board, pos board.Position) ([]byte, error) {
	tpl, err := web.Files.ReadFile("index.html")
	if err != nil {
		return nil, fmt.Errorf("httpapi: read page template: %w", err)
	}
	out := bytes.Replace(tpl, []byte(boardMarker), []byte(b.RenderSVG(pos)), 1)
	return out, nil
}
