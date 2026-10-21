package httpapi

import (
	"net/http"
	"time"

	"github.com/taolicus/komikan/internal/store"
)

// seatInfo describes one seat of a room.
type seatInfo struct {
	Seat     int    `json:"seat"`
	PlayerID string `json:"playerId"`
	Role     string `json:"role,omitempty"`
}

// roomSummary is the JSON shape of a room.
type roomSummary struct {
	ID        string     `json:"id"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	Seats     []seatInfo `json:"seats"`
	// Game is the state of the game in the room.
	Game *gameView `json:"game,omitempty"`
	// Board is the topology of the board, so a client can draw the very
	// graph the rules engine uses.
	Board *boardView `json:"board,omitempty"`
	// You describes the seat of the requesting player, if it has one.
	You *seatInfo `json:"you,omitempty"`
	// LegalMoves are the moves the requesting player may play, when it is
	// their turn.
	LegalMoves []moveView `json:"legalMoves,omitempty"`
	// URL is the shareable path of the room.
	URL string `json:"url"`
}

// roomList is the JSON shape of the room list.
type roomList struct {
	Rooms []roomSummary `json:"rooms"`
}

// handleCreateRoom creates a room with the creator in seat 1.
func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	player, ok := playerFrom(r.Context())
	if !ok {
		s.writeError(w, errForbidden)
		return
	}

	initial, err := encodeState(s.engine.NewGame())
	if err != nil {
		s.writeError(w, err)
		return
	}

	room, err := s.store.CreateRoom(r.Context(), store.NewID(), player.ID,
		store.State(`{}`), initial)
	if err != nil {
		s.writeError(w, err)
		return
	}

	s.writeJSON(w, http.StatusCreated, s.summarize(r, room, player.ID))
}

// handleJoinRoom seats the player in the free seat of a room.
func (s *Server) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	player, ok := playerFrom(r.Context())
	if !ok {
		s.writeError(w, errForbidden)
		return
	}

	roomID := r.PathValue("roomID")
	room, err := s.store.JoinRoom(r.Context(), roomID, player.ID)
	if err != nil {
		s.writeError(w, err)
		return
	}

	s.writeJSON(w, http.StatusOK, s.summarize(r, room, player.ID))
}

// handleListRooms lists the rooms the requesting player is seated in.
func (s *Server) handleListRooms(w http.ResponseWriter, r *http.Request) {
	player, ok := playerFrom(r.Context())
	if !ok {
		s.writeError(w, errForbidden)
		return
	}

	rooms, err := s.store.RoomsForPlayer(r.Context(), player.ID)
	if err != nil {
		s.writeError(w, err)
		return
	}

	list := roomList{Rooms: make([]roomSummary, 0, len(rooms))}
	for _, room := range rooms {
		list.Rooms = append(list.Rooms, s.summarize(r, room, player.ID))
	}
	s.writeJSON(w, http.StatusOK, list)
}

// handleGetRoom returns the summary of a room to one of its players.
func (s *Server) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	player, ok := playerFrom(r.Context())
	if !ok {
		s.writeError(w, errForbidden)
		return
	}

	room, err := s.store.GetRoom(r.Context(), r.PathValue("roomID"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	if _, seated := room.Seat(player.ID); !seated {
		s.writeError(w, errForbidden)
		return
	}

	s.writeJSON(w, http.StatusOK, s.summarize(r, room, player.ID))
}
