// Package httpapi serves the HTTP interface of the Komikan application.
//
// The package owns routing and request handling. Game logic lives in
// internal/rules, persistence in internal/store, and the board data in
// internal/board; none of them know about HTTP.
package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/taolicus/komikan/internal/board"
	"github.com/taolicus/komikan/internal/rules"
	"github.com/taolicus/komikan/internal/store"
	"github.com/taolicus/komikan/web"
)

// boardMarker is replaced in the page template by the rendered board.
const boardMarker = "<!--BOARD-->"

// errForbidden is returned when a player asks for something that is not
// theirs.
var errForbidden = errors.New("httpapi: forbidden")

// Config holds the dependencies of the server.
type Config struct {
	// Logger receives request and error logs. A nil logger discards them.
	Logger *slog.Logger
	// Version is the build version, reported by the version endpoint.
	Version string
	// Store persists players, rooms, and games.
	Store store.Store
	// Ruleset is the variant every room plays. A zero value means the
	// default ruleset.
	Ruleset rules.Ruleset
}

// Server holds the state shared by the handlers.
type Server struct {
	log           *slog.Logger
	board         *board.Board
	engine        *rules.Engine
	startPosition board.Position
	page          []byte
	roomsPage     []byte
	version       string
	store         store.Store
	hub           *eventHub
	mux           *http.ServeMux
}

// New builds the server and its routes.
func New(cfg Config) (*Server, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if cfg.Store == nil {
		return nil, errors.New("httpapi: a store is required")
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
	roomsPage, err := web.Files.ReadFile("rooms.html")
	if err != nil {
		return nil, fmt.Errorf("httpapi: read rooms page: %w", err)
	}

	ruleset := cfg.Ruleset
	if ruleset == (rules.Ruleset{}) {
		ruleset = rules.DefaultRuleset()
	}
	engine, err := rules.NewEngine(b, ruleset)
	if err != nil {
		return nil, fmt.Errorf("httpapi: build the rules engine: %w", err)
	}

	s := &Server{
		log:           logger,
		board:         b,
		engine:        engine,
		startPosition: pos,
		page:          page,
		roomsPage:     roomsPage,
		version:       cfg.Version,
		store:         cfg.Store,
		hub:           newEventHub(),
		mux:           http.NewServeMux(),
	}
	s.routes()
	return s, nil
}

// Handler returns the HTTP handler of the server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.Handle("GET /favicon.svg", http.FileServer(http.FS(web.Files)))

	s.mux.HandleFunc("POST /api/rooms", s.withIdentity(s.handleCreateRoom))
	s.mux.HandleFunc("POST /api/rooms/{roomID}/join", s.withIdentity(s.handleJoinRoom))
	s.mux.HandleFunc("GET /api/rooms", s.withIdentity(s.handleListRooms))
	s.mux.HandleFunc("GET /api/rooms/{roomID}", s.withIdentity(s.handleGetRoom))
	s.mux.HandleFunc("GET /api/rooms/{roomID}/board", s.withIdentity(s.handleRoomBoard))
	s.mux.HandleFunc("GET /api/rooms/{roomID}/events", s.withIdentity(s.handleRoomEvents))
	s.mux.HandleFunc("POST /api/rooms/{roomID}/moves", s.withIdentity(s.handleSubmitMove))

	// The room pages. They are served to anyone: the API behind them checks
	// who is allowed to see a room.
	s.mux.HandleFunc("GET /rooms/{$}", s.handleRoomsPage)
	s.mux.HandleFunc("GET /rooms/{roomID}", s.handleRoomsPage)

	// "/{$}" matches the root path exactly; a bare "/" would match every
	// path and swallow the routes added later.
	s.mux.HandleFunc("GET /{$}", s.handleRoot)
}

// handleRoomsPage serves the rooms page, which the browser fills in from the
// API.
func (s *Server) handleRoomsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if _, err := w.Write(s.roomsPage); err != nil {
		s.log.Error("write rooms page", "error", err)
	}
}

// handleRoot serves the coming-soon page. Other paths are left to the tasks
// that add them, such as the room and game pages.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
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

// writeJSON writes a JSON response.
func (s *Server) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		s.log.Error("write json", "error", err)
	}
}

// writeError maps a domain error onto a status code and writes it.
//
// The body always carries an "error" message and a machine-readable "code",
// so a client can tell a stale version from an illegal move without reading
// the message.
func (s *Server) writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	message := "internal error"

	var apiErr *apiError
	switch {
	case errors.As(err, &apiErr):
		status = apiErr.status
		code = apiErr.code
		message = apiErr.msg
	case errors.Is(err, store.ErrNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "not found"
	case errors.Is(err, store.ErrConflict):
		status, code, message = http.StatusConflict, "CONFLICT", "conflict"
	case errors.Is(err, store.ErrInvalid):
		status, code, message = http.StatusBadRequest, "MALFORMED", "malformed request"
	case errors.Is(err, errForbidden):
		status, code, message = http.StatusForbidden, "FORBIDDEN", "not your room"
	case errors.Is(err, errCSRF):
		status, code, message = http.StatusForbidden, "CSRF_FAILED", "csrf check failed"
	}

	if status == http.StatusInternalServerError {
		s.log.Error("request failed", "error", err)
	}
	s.writeJSON(w, status, map[string]string{"error": message, "code": code})
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
